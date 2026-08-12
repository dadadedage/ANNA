import "./style.css";
import { NotesApp } from "./app/app";
import { connectAnna, NotesGateway, SummaryGateway } from "./host/anna";

const root = document.querySelector<HTMLElement>("#app");
if (!root) throw new Error("App root is missing.");
const appRoot = root;

void start();

async function start(): Promise<void> {
  try {
    const anna = await connectAnna();
    await new NotesApp(appRoot, new NotesGateway(anna), new SummaryGateway(anna)).start();
  } catch (error) {
    appRoot.textContent = error instanceof Error ? `Unable to connect to Anna: ${error.message}` : "Unable to connect to Anna.";
  }
}
