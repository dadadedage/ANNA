import type { Note } from "../domain/notes";

const NOTES_KEY = "mini-notes.notes";
export const TOOL_ID = "tool-local-anna-mini-notes";
export const SUMMARIZE_METHOD = "summarize";

type AnnaStorage = {
  get(input: { key: string }): Promise<{ value?: unknown }>;
  set(input: { key: string; value: unknown }): Promise<void>;
};

type AnnaTools = {
  invoke(input: { tool_id: string; method: string; args: unknown }): Promise<unknown>;
};

export type AnnaHost = { storage: AnnaStorage; tools: AnnaTools };

export async function connectAnna(): Promise<AnnaHost> {
  const moduleUrl = "/static/anna-apps/_sdk/latest/index.js";
  const runtime = (await import(/* @vite-ignore */ moduleUrl)) as {
    AnnaAppRuntime: { connect(): Promise<AnnaHost> };
  };
  return runtime.AnnaAppRuntime.connect();
}

export class NotesGateway {
  constructor(private readonly anna: AnnaHost) {}

  async load(): Promise<Note[]> {
    const { value } = await this.anna.storage.get({ key: NOTES_KEY });
    if (!value || typeof value !== "object" || !Array.isArray((value as { notes?: unknown }).notes)) {
      return [];
    }
    return (value as { notes: Note[] }).notes;
  }

  save(notes: Note[]): Promise<void> {
    return this.anna.storage.set({ key: NOTES_KEY, value: { notes } });
  }
}

export class SummaryGateway {
  constructor(private readonly anna: AnnaHost) {}

  async summarize(notes: Note[]): Promise<string> {
    const result = await this.anna.tools.invoke({
      tool_id: TOOL_ID,
      method: SUMMARIZE_METHOD,
      args: { notes },
    });
    if (!result || typeof result !== "object" || typeof (result as { summary?: unknown }).summary !== "string") {
      throw new Error("The summarize tool returned an invalid response.");
    }
    return (result as { summary: string }).summary;
  }
}
