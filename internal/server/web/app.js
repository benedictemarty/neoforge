// Application neoforge : éditeur Monaco (NeoBASIC) + émulateur Phosphoneo (WASM).
import { registerNeoBasic } from "/neobasic-lang.js";
import { decodeBase64, errorLine, storageName, filterHelp } from "/editor-logic.js";

const el = (id) => document.getElementById(id);
const status = (msg, err) => { const s = el("status"); s.textContent = msg; s.classList.toggle("err", !!err); };
const diag = (msg, err) => { const d = el("diag"); d.textContent = msg; d.classList.toggle("err", !!err); };

const DEFAULT_SOURCE = `' neoforge - premier programme NeoBASIC
cls
print "Bonjour depuis neoforge !"
for i = 1 to 10
  print "ligne "; i
next
`;

// ─── Émulateur Phosphoneo (WebAssembly) ─────────────────────────────────────
// Module global attendu par phosphoneo.js (build emscripten classique) ; les
// fichiers .js/.wasm/.data sont servis sous /emu/ depuis NEOFORGE_PHOSPHONEO_WEB.
let emuReady = false;
window.Module = {
  canvas: el("canvas"),
  arguments: ["--sdl", "--scale", "1"],
  // .wasm et .data sont résolus relativement à la page par emscripten : les renvoyer sous /emu/.
  locateFile: (path) => "/emu/" + path.split("/").pop(),
  print: (t) => console.log(t),
  printErr: (t) => { console.log(t); if (/^phosphoneo|^sdl/.test(t)) el("emu-status").textContent = t; },
  setStatus: (t) => { if (t) el("emu-status").textContent = t; },
  onRuntimeInitialized: () => { emuReady = true; el("emu-status").textContent = "NeoBASIC démarre…"; },
};

async function loadEmulator(cfg) {
  if (!cfg.emulator) {
    el("emu-status").textContent = "Émulateur absent : construire Phosphoneo (make wasm) ou régler NEOFORGE_PHOSPHONEO_WEB.";
    return;
  }
  const s = document.createElement("script");
  s.src = "/emu/phosphoneo.js";
  s.onerror = () => { el("emu-status").textContent = "Échec du chargement de /emu/phosphoneo.js"; };
  document.body.appendChild(s);
}

// runInEmulator : écrit le .bas dans /storage et le lance (load "…" + run).
function runInEmulator(bas, name) {
  if (!emuReady) { status("Émulateur non prêt", true); return; }
  const path = "/storage/" + storageName(name);
  Module.FS.writeFile(path, bas);
  const rc = Module.ccall("web_load_neo", "number", ["string"], [path]);
  if (rc === 0) { status(path.slice(9) + " lancé"); Module.canvas.focus(); }
  else status("échec du chargement dans l'émulateur", true);
}

// ─── Fichiers ───────────────────────────────────────────────────────────────
async function refreshFiles() {
  const names = await fetch("/api/files").then((r) => r.json());
  const sel = el("files");
  sel.innerHTML = "";
  for (const n of names) { const o = document.createElement("option"); o.value = o.textContent = n; sel.appendChild(o); }
}

async function main() {
  const cfg = await fetch("/api/config").then((r) => r.json());
  const keywords = await fetch("/api/keywords").then((r) => r.json());
  const help = await fetch("/help.json").then((r) => r.json()).then((j) => j.entries).catch(() => []);
  loadEmulator(cfg);
  refreshFiles();

  require.config({ paths: { vs: "/vendor/vs" } });
  self.MonacoEnvironment = {
    getWorkerUrl() {
      const base = location.origin + "/vendor/vs";
      const src = "self.MonacoEnvironment={baseUrl:'" + base + "/'};importScripts('" + base + "/base/worker/workerMain.js');";
      return URL.createObjectURL(new Blob([src], { type: "text/javascript" }));
    },
  };

  require(["vs/editor/editor.main"], () => {
    registerNeoBasic(monaco, keywords, help);
    const editor = monaco.editor.create(el("editor"), {
      value: DEFAULT_SOURCE, language: "neobasic", theme: "vs-dark", fontSize: 15,
      minimap: { enabled: false }, automaticLayout: true,
    });
    status("neoforge " + cfg.version);

    // build : tokenise le source ; marque la ligne en erreur dans l'éditeur.
    async function build() {
      const r = await fetch("/api/build", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ source: editor.getValue() }) });
      const j = await r.json();
      const model = editor.getModel();
      if (j.error) {
        const ln = errorLine(j.error);
        monaco.editor.setModelMarkers(model, "neobas", ln ? [{ startLineNumber: ln, endLineNumber: ln, startColumn: 1, endColumn: model.getLineMaxColumn(ln), message: j.error, severity: monaco.MarkerSeverity.Error }] : []);
        diag(j.error, true);
        return null;
      }
      monaco.editor.setModelMarkers(model, "neobas", []);
      const bas = decodeBase64(j.bas);
      diag(j.lines + " lignes, " + bas.length + " octets tokenisés");
      return bas;
    }

    const run = async () => { const bas = await build(); if (bas) runInEmulator(bas, el("fname").value); };
    el("btn-run").addEventListener("click", run);
    editor.addCommand(monaco.KeyCode.F5, run);

    el("btn-download").addEventListener("click", async () => {
      const bas = await build();
      if (!bas) return;
      const a = document.createElement("a");
      a.href = URL.createObjectURL(new Blob([bas], { type: "application/octet-stream" }));
      a.download = storageName(el("fname").value);
      a.click();
    });

    const save = async () => {
      const name = el("fname").value.trim();
      if (!name) { status("Indiquer un nom (nom.bsc)", true); return; }
      const r = await fetch("/api/file?name=" + encodeURIComponent(name), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ source: editor.getValue() }) });
      const j = await r.json();
      if (j.error) status(j.error, true); else { status(name + " enregistré"); refreshFiles(); }
    };
    el("btn-save").addEventListener("click", save);
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, save);

    el("btn-open").addEventListener("click", async () => {
      const name = el("files").value;
      if (!name) return;
      const j = await fetch("/api/file?name=" + encodeURIComponent(name)).then((r) => r.json());
      if (j.error) { status(j.error, true); return; }
      editor.setValue(j.source);
      el("fname").value = j.name;
      status(name + " ouvert");
    });

    // Importer un .bas : détokenisé côté serveur (/api/detok), ouvert dans l'éditeur.
    el("import-bas").addEventListener("change", async (ev) => {
      const f = ev.target.files[0];
      if (!f) return;
      const j = await fetch("/api/detok", { method: "POST", body: await f.arrayBuffer() }).then((r) => r.json());
      if (j.error) { status(j.error, true); return; }
      editor.setValue(j.source);
      el("fname").value = f.name.replace(/\.bas$/i, ".bsc");
      status(f.name + " détokenisé");
      ev.target.value = "";
    });

    // ─── Panneau d'aide (F1) : liste filtrable, clic = insertion de la syntaxe ─────
    const renderHelp = () => {
      const list = el("help-list");
      list.innerHTML = "";
      for (const h of filterHelp(help, el("help-filter").value)) {
        const d = document.createElement("div");
        d.className = "help-item";
        d.innerHTML = '<span class="section"></span><div class="syntax"></div><div class="notes"></div>';
        d.querySelector(".section").textContent = h.section;
        d.querySelector(".syntax").textContent = h.syntax;
        d.querySelector(".notes").textContent = h.notes;
        d.addEventListener("click", () => { editor.trigger("help", "type", { text: h.name }); editor.focus(); });
        list.appendChild(d);
      }
    };
    const toggleHelp = (show) => {
      const panel = el("help");
      panel.hidden = show === undefined ? !panel.hidden : !show;
      document.querySelector("main").classList.toggle("with-help", !panel.hidden);
      if (!panel.hidden) { renderHelp(); el("help-filter").focus(); }
    };
    el("btn-help").addEventListener("click", () => toggleHelp());
    el("help-close").addEventListener("click", () => toggleHelp(false));
    el("help-filter").addEventListener("input", renderHelp);
    editor.addCommand(monaco.KeyCode.F1, () => toggleHelp(true));

    el("btn-new").addEventListener("click", () => { editor.setValue(DEFAULT_SOURCE); el("fname").value = ""; diag(""); });
    el("btn-focus").addEventListener("click", () => Module.canvas.focus());
    el("btn-fullscreen").addEventListener("click", () => Module.canvas.requestFullscreen && Module.canvas.requestFullscreen());
  });
}

main().catch((e) => status(String(e), true));
