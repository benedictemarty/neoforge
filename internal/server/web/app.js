// Application neoforge : éditeur Monaco (NeoBASIC) + émulateur Phosphoneo (WASM).
import { registerNeoBasic } from "/neobasic-lang.js";
import { decodeBase64, errorLine, storageName, filterHelp, tabsAdd, tabsActivate, tabsClose, tabsFindByName, tabTitle } from "/editor-logic.js";
import { setupGfxEditor } from "/gfx-editor.js";
import { setupDebugger } from "/debugger.js";

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
  // Trinity : NeoDOS est résident, NeoBASIC est lancé depuis /storage/boot/auto.txt → on y place
  // le binaire NeoBASIC (servi depuis NEOFORGE_NEOBASIC_BIN) avant le démarrage du firmware.
  if (cfg.neobasic) {
    const bin = new Uint8Array(await fetch("/emu-boot/neobasic.bin").then((r) => r.arrayBuffer()));
    Module.preRun = [() => {
      Module.FS.mkdirTree("/storage/boot");
      Module.FS.writeFile("/storage/boot/neobasic.bin", bin);
      Module.FS.writeFile("/storage/boot/auto.txt", "neobasic.bin\n");
    }];
  } else {
    el("emu-status").textContent = "NeoBASIC introuvable (NEOFORGE_NEOBASIC_BIN) : l'émulateur démarrera sur NeoDOS.";
  }
  const s = document.createElement("script");
  s.src = "/emu/phosphoneo.js";
  s.onerror = () => { el("emu-status").textContent = "Échec du chargement de /emu/phosphoneo.js"; };
  document.body.appendChild(s);
}

// sendKey : touche synthétique vers le canvas SDL (Échap = Break de NeoBASIC).
function sendKey(key, code, keyCode) {
  const c = Module.canvas;
  c.focus();
  for (const type of ["keydown", "keyup"]) c.dispatchEvent(new KeyboardEvent(type, { key, code, keyCode, which: keyCode, bubbles: true, cancelable: true }));
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
      model: null, language: "neobasic", theme: "vs-dark", fontSize: 15,
      minimap: { enabled: false }, automaticLayout: true,
    });
    status("neoforge " + cfg.version);

    // ─── Onglets : un modèle Monaco par programme, « ● » si non enregistré (S1-3) ──
    let tabs = { tabs: [], activeId: null };
    const models = new Map();
    let seq = 0;
    const activeTab = () => tabs.tabs.find((t) => t.id === tabs.activeId);
    const renderTabs = () => {
      const bar = el("tabs");
      bar.innerHTML = "";
      for (const t of tabs.tabs) {
        const d = document.createElement("div");
        d.className = "tab" + (t.id === tabs.activeId ? " active" : "");
        d.setAttribute("role", "tab");
        d.dataset.id = t.id;
        const label = document.createElement("span");
        label.textContent = tabTitle(t, models.get(t.id).getAlternativeVersionId());
        const close = document.createElement("span");
        close.className = "close";
        close.textContent = "✕";
        close.title = "Fermer";
        close.addEventListener("click", (ev) => { ev.stopPropagation(); closeTab(t.id); });
        d.append(label, close);
        d.addEventListener("click", () => activateTab(t.id));
        bar.appendChild(d);
      }
    };
    const activateTab = (id) => {
      tabs = tabsActivate(tabs, id);
      const t = activeTab();
      editor.setModel(t ? models.get(t.id) : null);
      el("fname").value = t ? t.name : "";
      renderTabs();
      editor.focus();
    };
    const openTab = (name, source) => {
      const existing = name && tabsFindByName(tabs, name);
      if (existing) { models.get(existing.id).setValue(source); existing.savedVersion = models.get(existing.id).getAlternativeVersionId(); activateTab(existing.id); return; }
      const id = ++seq;
      const model = monaco.editor.createModel(source, "neobasic");
      model.onDidChangeContent(() => renderTabs());
      models.set(id, model);
      tabs = tabsAdd(tabs, { id, name: name || "", savedVersion: model.getAlternativeVersionId() });
      activateTab(id);
    };
    const closeTab = (id) => {
      const t = tabs.tabs.find((x) => x.id === id);
      if (t && models.get(id).getAlternativeVersionId() !== t.savedVersion && !confirm("Fermer « " + (t.name || "sans titre") + " » sans enregistrer ?")) return;
      tabs = tabsClose(tabs, id);
      models.get(id).dispose();
      models.delete(id);
      if (!tabs.tabs.length) openTab("", DEFAULT_SOURCE); else activateTab(tabs.activeId);
    };
    el("fname").addEventListener("input", () => { const t = activeTab(); if (t) { t.name = el("fname").value.trim(); renderTabs(); } });
    openTab("", DEFAULT_SOURCE);

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

    // ⚙ Compiler : /api/compile → .neo dans /storage → web_load_neo (NeoBASIC tape load "prog.neo").
    const compile = async () => {
      const r = await fetch("/api/compile", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ source: editor.getValue() }) });
      const j = await r.json();
      const model = editor.getModel();
      if (j.error) {
        const ln = errorLine(j.error);
        monaco.editor.setModelMarkers(model, "neoforgec", ln ? [{ startLineNumber: ln, endLineNumber: ln, startColumn: 1, endColumn: model.getLineMaxColumn(ln), message: j.error, severity: monaco.MarkerSeverity.Error }] : []);
        diag(j.error, true);
        return;
      }
      monaco.editor.setModelMarkers(model, "neoforgec", []);
      diag("compilé : " + j.bytes + " octets (.neo), " + (j.symbols || []).length + " variable(s)");
      dbg.setSymbols(j.symbols, j.labels);
      if (!emuReady) { status("Émulateur non prêt", true); return; }
      const path = "/storage/" + storageName(el("fname").value).replace(/\.bas$/, ".neo");
      Module.FS.writeFile(path, decodeBase64(j.neo));
      const rc = Module.ccall("web_load_neo", "number", ["string"], [path]);
      if (rc === 0) { status(path.slice(9) + " compilé et lancé"); Module.canvas.focus(); } else status("échec du lancement", true);
    };
    el("btn-compile").addEventListener("click", compile);
    editor.addCommand(monaco.KeyCode.F6, compile);

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
      if (j.error) { status(j.error, true); return; }
      const t = activeTab();
      if (t) { t.name = name; t.savedVersion = editor.getModel().getAlternativeVersionId(); renderTabs(); }
      status(name + " enregistré");
      refreshFiles();
    };
    el("btn-save").addEventListener("click", save);
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, save);

    el("btn-open").addEventListener("click", async () => {
      const name = el("files").value;
      if (!name) return;
      const j = await fetch("/api/file?name=" + encodeURIComponent(name)).then((r) => r.json());
      if (j.error) { status(j.error, true); return; }
      openTab(j.name, j.source);
      status(name + " ouvert");
    });

    // Importer un .bas : détokenisé côté serveur (/api/detok), ouvert dans l'éditeur.
    el("import-bas").addEventListener("change", async (ev) => {
      const f = ev.target.files[0];
      if (!f) return;
      const j = await fetch("/api/detok", { method: "POST", body: await f.arrayBuffer() }).then((r) => r.json());
      if (j.error) { status(j.error, true); return; }
      openTab(f.name.replace(/\.bas$/i, ".bsc"), j.source);
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

    // Fichiers de données du programme (graphismes .gfx, niveaux, scores…) → /storage de l'émulateur.
    el("storage-files").addEventListener("change", async (ev) => {
      const files = Array.from(ev.target.files);
      if (!files.length || !emuReady) { if (!emuReady) status("Émulateur non prêt", true); return; }
      for (const f of files) Module.FS.writeFile("/storage/" + storageName(f.name, true), new Uint8Array(await f.arrayBuffer()));
      status(files.length + " fichier(s) dans /storage : " + files.map((f) => storageName(f.name, true)).join(", "));
      ev.target.value = "";
    });

    el("btn-new").addEventListener("click", () => { openTab("", DEFAULT_SOURCE); diag(""); });

    // ─── 🎨 Graphismes : panneau à la place de l'éditeur (S3-1, S3-2) ────────
    const sendToStorage = (name, bytes) => {
      if (!emuReady) { status("Émulateur non prêt", true); return; }
      Module.FS.writeFile("/storage/" + storageName(name, true), bytes);
      status(storageName(name, true) + " envoyé dans /storage");
    };
    setupGfxEditor({ status, sendToStorage, insertText: (text) => { editor.trigger("gfx", "type", { text }); toggleGfx(false); editor.focus(); } });
    const toggleGfx = (show) => {
      const pane = el("gfx-pane");
      pane.hidden = show === undefined ? !pane.hidden : !show;
      el("editor-pane").hidden = !pane.hidden;
    };
    el("btn-gfx").addEventListener("click", () => toggleGfx());

    // ─── 🐞 Débogueur (S5-4) ─────────────────────────────────────────────────
    const dbg = setupDebugger({ status, isReady: () => emuReady });
    el("btn-dbg").addEventListener("click", () => { el("dbg").hidden = !el("dbg").hidden; });
    el("dbg-close").addEventListener("click", () => { el("dbg").hidden = true; });
    el("btn-focus").addEventListener("click", () => Module.canvas.focus());
    // Stop : web_type("\\e") (frappe automatique de Phosphoneo) si l'export existe, sinon touche synthétique.
    el("btn-stop").addEventListener("click", () => {
      if (!emuReady) return;
      if (Module._web_type) Module.ccall("web_type", "number", ["string"], ["\\e"]); else sendKey("Escape", "Escape", 27);
      status("Break envoyé");
    });
    el("btn-reset").addEventListener("click", () => {
      if (!emuReady) return;
      if (!Module._web_reset) { status("Reset indisponible : reconstruire Phosphoneo (make wasm)", true); return; }
      Module.ccall("web_reset", null, [], []);
      status("Émulateur redémarré");
    });
    el("btn-fullscreen").addEventListener("click", () => Module.canvas.requestFullscreen && Module.canvas.requestFullscreen());
  });
}

main().catch((e) => status(String(e), true));
