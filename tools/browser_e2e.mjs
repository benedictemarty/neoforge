// Validation navigateur de bout en bout (S1-2) : lance Chrome headless piloté par CDP,
// ouvre neoforge, attend l'émulateur, clique ▶ Exécuter, puis capture l'écran et lit
// vérifie le statut « lancé » (le chargement a été tapé à l'invite NeoBASIC) ; la capture
// PNG montre l'écran de l'émulateur pour contrôle visuel.
//   node tools/browser_e2e.mjs [http://127.0.0.1:8098] [capture.png]
import { spawn } from "node:child_process";
import { writeFileSync } from "node:fs";

const url = process.argv[2] || "http://127.0.0.1:8098/";
const shot = process.argv[3] || "/tmp/neoforge-browser.png";
const port = 9333;
const chrome = spawn(process.env.CHROME || "google-chrome", [
  "--headless=new", "--no-sandbox", "--disable-gpu", "--enable-unsafe-swiftshader", "--autoplay-policy=no-user-gesture-required",
  "--window-size=1400,900", `--remote-debugging-port=${port}`, "about:blank",
], { stdio: "ignore" });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const fail = (msg) => { console.error("ÉCHEC :", msg); chrome.kill(); process.exit(1); };

try {
  let targets;
  for (let i = 0; i < 50 && !targets; i++) {
    await sleep(200);
    targets = await fetch(`http://127.0.0.1:${port}/json`).then((r) => r.json()).catch(() => null);
  }
  const page = targets.find((t) => t.type === "page");
  const ws = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise((r) => (ws.onopen = r));
  let id = 0; const pending = new Map(); const logs = [];
  ws.onmessage = (ev) => {
    const m = JSON.parse(ev.data);
    if (m.id && pending.has(m.id)) { pending.get(m.id)(m.result); pending.delete(m.id); }
    if (m.method === "Runtime.consoleAPICalled") logs.push(m.params.args.map((a) => a.value).join(" "));
  };
  const send = (method, params = {}) => new Promise((r) => { pending.set(++id, r); ws.send(JSON.stringify({ id, method, params })); });
  const evaluate = async (expression) => (await send("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true })).result?.value;

  await send("Runtime.enable");
  await send("Page.enable");
  await send("Page.navigate", { url });
  // Attendre que l'émulateur soit prêt (message « NeoBASIC démarre… » puis invite ≈ 2 s).
  let ready = false;
  for (let i = 0; i < 60 && !ready; i++) { await sleep(500); ready = await evaluate("!!(window.Module && Module.FS && Module.ccall)"); }
  if (!ready) fail("émulateur WASM non initialisé (" + logs.slice(-5).join(" | ") + ")");
  await sleep(4000);
  await evaluate(`document.getElementById("btn-run").click()`);
  await sleep(6000);
  const status = await evaluate(`document.getElementById("status").textContent`);
  const diag = await evaluate(`document.getElementById("diag").textContent`);
  const png = (await send("Page.captureScreenshot", { format: "png" })).data;
  writeFileSync(shot, Buffer.from(png, "base64"));
  console.log("statut :", status, "|", diag, "| capture :", shot);
  console.log(logs.filter((l) => /phosphoneo|chargement|FIS|sdl/.test(l)).slice(-6).join("\n"));
  if (!/lancé/.test(status || "")) fail("le programme n'a pas été lancé : " + status);
  chrome.kill();
} catch (e) { fail(String(e)); }
