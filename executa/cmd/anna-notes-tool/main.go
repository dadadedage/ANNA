package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

const toolName = "summarize"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type invokeParams struct {
	Tool      string `json:"tool"`
	Arguments struct {
		Notes []note `json:"notes"`
	} `json:"arguments"`
	InvokeID string `json:"invoke_id"`
}

type note struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Order   int    `json:"order"`
}

type server struct {
	writer    *bufio.Writer
	writeMu   sync.Mutex
	pending   map[string]chan response
	pendingMu sync.Mutex
	nextID    atomic.Uint64
	handlers  sync.WaitGroup
}

func main() {
	s := &server{writer: bufio.NewWriter(os.Stdout), pending: make(map[string]chan response)}
	s.run()
}

func (s *server) run() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var message request
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			s.write(response{JSONRPC: "2.0", ID: nil, Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if s.deliverReverseResponse(message) {
			continue
		}
		s.handlers.Add(1)
		go func() {
			defer s.handlers.Done()
			s.handle(message)
		}()
	}
	s.handlers.Wait()
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "stdin read error:", err)
	}
}

func (s *server) handle(message request) {
	switch message.Method {
	case "initialize":
		s.reply(message.ID, map[string]interface{}{
			"protocolVersion":      "2.0",
			"client_capabilities": map[string]interface{}{"sampling": map[string]interface{}{}},
			// `capabilities` keeps the current local CLI v0.1.46 compatible.
			"capabilities": map[string]interface{}{"sampling": map[string]interface{}{}},
		})
	case "describe":
		s.reply(message.ID, manifest())
	case "invoke":
		var params invokeParams
		if err := json.Unmarshal(message.Params, &params); err != nil {
			s.fail(message.ID, -32602, "invalid invoke parameters")
			return
		}
		if params.Tool != toolName {
			s.fail(message.ID, -32602, "unknown tool")
			return
		}
		summary, err := s.sample(params.InvokeID, params.Arguments.Notes)
		if err != nil {
			s.fail(message.ID, -32603, err.Error())
			return
		}
		s.reply(message.ID, map[string]string{"summary": summary})
	case "health":
		s.reply(message.ID, map[string]bool{"ok": true})
	case "shutdown":
		s.reply(message.ID, map[string]bool{"ok": true})
	default:
		s.fail(message.ID, -32601, "method not found")
	}
}

func (s *server) sample(invokeID string, notes []note) (string, error) {
	id := strconv.FormatUint(s.nextID.Add(1), 10)
	waiter := make(chan response, 1)
	s.pendingMu.Lock()
	s.pending[id] = waiter
	s.pendingMu.Unlock()
	defer func() {
		s.pendingMu.Lock()
		delete(s.pending, id)
		s.pendingMu.Unlock()
	}()

	content, _ := json.Marshal(notes)
	s.write(request{
		JSONRPC: "2.0", ID: json.RawMessage(id), Method: "sampling/createMessage",
		Params: mustJSON(map[string]interface{}{
			"messages": []map[string]string{{"role": "user", "content": "Summarize these notes concisely:\n" + string(content)}},
			"metadata": map[string]string{"invoke_id": invokeID},
		}),
	})

	result := <-waiter
	if result.Error != nil {
		return "", fmt.Errorf("sampling failed: %s", result.Error.Message)
	}
	text, ok := extractText(result.Result)
	if !ok {
		return "", fmt.Errorf("sampling returned no text")
	}
	return text, nil
}

func (s *server) deliverReverseResponse(message request) bool {
	if message.Method != "" || len(message.ID) == 0 {
		return false
	}
	id := string(message.ID)
	s.pendingMu.Lock()
	waiter, found := s.pending[id]
	s.pendingMu.Unlock()
	if !found {
		return false
	}
	var result interface{}
	if len(message.Result) > 0 {
		if err := json.Unmarshal(message.Result, &result); err != nil {
			result = nil
		}
	}
	waiter <- response{JSONRPC: "2.0", ID: message.ID, Result: result, Error: message.Error}
	return true
}

func (s *server) reply(id json.RawMessage, result interface{}) {
	s.write(response{JSONRPC: "2.0", ID: id, Result: result})
}
func (s *server) fail(id json.RawMessage, code int, message string) {
	s.write(response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}})
}
func (s *server) write(value interface{}) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if err := json.NewEncoder(s.writer).Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, "stdout write error:", err)
		return
	}
	if err := s.writer.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "stdout flush error:", err)
	}
}

func manifest() map[string]interface{} {
	return map[string]interface{}{
		"name": "tool-local-anna-mini-notes", "display_name": "Anna Mini Notes", "version": "0.1.0",
		"description": "Summarizes Mini Notes through host LLM sampling.", "host_capabilities": []string{"llm.sample", "llm.complete"}, "runtime": "go",
		"tools": []map[string]interface{}{{"name": toolName, "description": "Summarize notes.", "parameters": []map[string]interface{}{{"name": "notes", "type": "array", "required": true, "description": "Ordered notes to summarize."}}}},
	}
}

func extractText(value interface{}) (string, bool) {
	result, ok := value.(map[string]interface{})
	if !ok {
		return "", false
	}
	if text, ok := result["text"].(string); ok {
		return text, true
	}
	if content, ok := result["content"].(map[string]interface{}); ok {
		if text, ok := content["text"].(string); ok {
			return text, true
		}
	}
	if content, ok := result["content"].([]interface{}); ok {
		for _, item := range content {
			if block, ok := item.(map[string]interface{}); ok {
				if text, ok := block["text"].(string); ok {
					return text, true
				}
			}
		}
	}
	return "", false
}

func mustJSON(value interface{}) json.RawMessage { bytes, _ := json.Marshal(value); return bytes }
