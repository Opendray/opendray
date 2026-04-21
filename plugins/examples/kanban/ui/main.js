// kanban/ui/main.js — M2 reference plugin.
// Exercises the full bridge surface: storage.set/get, workbench.showMessage,
// events.subscribe. The SDK shim at window.opendray is injected by the
// Flutter webview host before this script runs.

const STORAGE_KEY = "cards";
let cards = [];

async function loadCards() {
  cards = await opendray.storage.get(STORAGE_KEY, []);
  render();
}

async function saveCards() {
  await opendray.storage.set(STORAGE_KEY, cards);
}

function render() {
  for (const col of document.querySelectorAll(".column")) {
    const list = col.querySelector("ul");
    list.innerHTML = "";
    for (const c of cards.filter(c => c.status === col.dataset.status)) {
      const li = document.createElement("li");
      li.textContent = c.title;
      li.addEventListener("click", () => removeCard(c.id));
      list.appendChild(li);
    }
  }
}

async function addCard() {
  const title = "Card " + Math.floor(Math.random() * 1000);
  cards.push({ id: crypto.randomUUID(), title, status: "todo" });
  await saveCards();
  render();
  await opendray.workbench.showMessage(`Added "${title}"`, { kind: "info" });
}

async function removeCard(id) {
  cards = cards.filter(c => c.id !== id);
  await saveCards();
  render();
}

document.getElementById("addCard").addEventListener("click", addCard);

opendray.events.subscribe("session.idle", () => {
  const el = document.getElementById("sessionStatus");
  el.textContent = "a session is idle";
  el.hidden = false;
  setTimeout(() => { el.hidden = true; }, 3000);
});

loadCards();
