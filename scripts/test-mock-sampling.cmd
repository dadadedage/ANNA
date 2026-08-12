@echo off
setlocal
if exist "C:\Program Files\Go\bin\go.exe" set "PATH=C:\Program Files\Go\bin;%PATH%"
npx anna-app executa dev --dir executa --mock-sampling fixtures\sampling.jsonl --invoke summarize --args "{\"notes\":[{\"id\":\"note-1\",\"content\":\"Follow up with the client\",\"order\":1}]}" --json
