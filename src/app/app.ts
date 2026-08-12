import { addNote, removeNote, sortNotes, type Note } from "../domain/notes";
import { NotesGateway, SummaryGateway } from "../host/anna";

export class NotesApp {
  private notes: Note[] = [];

  constructor(
    private readonly root: HTMLElement,
    private readonly notesGateway: NotesGateway,
    private readonly summaryGateway: SummaryGateway,
  ) {}

  async start(): Promise<void> {
    this.notes = sortNotes(await this.notesGateway.load());
    this.render();
  }

  private render(message = ""): void {
    this.root.innerHTML = `
      <section class="notes-shell">
        <header><p class="eyebrow">ANNA APP</p><h1>Mini Notes</h1></header>
        <form id="note-form"><label for="note-input">New note</label><textarea id="note-input" rows="3" placeholder="Write a short note"></textarea><button type="submit">Save note</button></form>
        <p id="status" role="status">${escapeHtml(message)}</p>
        <ol id="notes-list">${this.notes.map((note) => `<li><span><b>${note.order}.</b> ${escapeHtml(note.content)}</span><button class="delete" data-note-id="${note.id}" aria-label="Delete note ${note.order}">Delete</button></li>`).join("") || "<li class=empty>No notes yet.</li>"}</ol>
        <button id="summarize" ${this.notes.length === 0 ? "disabled" : ""}>Summarize</button>
        <section id="summary" aria-live="polite"></section>
      </section>`;

    this.root.querySelector<HTMLFormElement>("#note-form")?.addEventListener("submit", (event) => void this.handleAdd(event));
    this.root.querySelectorAll<HTMLButtonElement>("[data-note-id]").forEach((button) => button.addEventListener("click", () => void this.handleRemove(button.dataset.noteId ?? "")));
    this.root.querySelector<HTMLButtonElement>("#summarize")?.addEventListener("click", () => void this.handleSummarize());
  }

  private async handleAdd(event: SubmitEvent): Promise<void> {
    event.preventDefault();
    const input = this.root.querySelector<HTMLTextAreaElement>("#note-input");
    try {
      this.notes = addNote(this.notes, input?.value ?? "", crypto.randomUUID());
      await this.notesGateway.save(this.notes);
      this.render("Saved.");
    } catch (error) {
      this.render(errorMessage(error));
    }
  }

  private async handleRemove(id: string): Promise<void> {
    this.notes = removeNote(this.notes, id);
    try {
      await this.notesGateway.save(this.notes);
      this.render("Deleted.");
    } catch (error) {
      this.render(errorMessage(error));
    }
  }

  private async handleSummarize(): Promise<void> {
    const target = this.root.querySelector<HTMLElement>("#summary");
    if (!target) return;
    target.textContent = "Summarizing...";
    try {
      target.textContent = await this.summaryGateway.summarize(await this.notesGateway.load());
    } catch (error) {
      target.textContent = errorMessage(error);
    }
  }
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>'"]/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "'": "&#39;", '"': "&quot;" })[character] ?? character);
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Something went wrong.";
}
