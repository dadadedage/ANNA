export type Note = {
  id: string;
  content: string;
  order: number;
};

export function addNote(notes: Note[], content: string, id: string): Note[] {
  const cleaned = content.trim();
  if (!cleaned) throw new Error("Enter a note before saving.");

  const nextOrder = notes.reduce((max, note) => Math.max(max, note.order), 0) + 1;
  return [...notes, { id, content: cleaned, order: nextOrder }];
}

export function removeNote(notes: Note[], id: string): Note[] {
  return notes.filter((note) => note.id !== id);
}

export function sortNotes(notes: Note[]): Note[] {
  return [...notes].sort((left, right) => left.order - right.order);
}
