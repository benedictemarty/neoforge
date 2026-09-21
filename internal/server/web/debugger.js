// Débogueur (S5-4) : pause / pas / reprise de Phosphoneo (exports web_pause…), registres,
// variables du programme compilé (symboles de /api/compile), mémoire.
import { decodeRegs, decodeNumber, decodeString, hexDump, hex4, nearestLabel } from "/editor-logic.js";

const el = (id) => document.getElementById(id);

export function setupDebugger({ status, isReady }) {
  let symbols = [], labels = {};
  let timer = null;

  const available = () => isReady() && typeof Module._web_pause === "function";
  const peek = (addr, n) => {
    const out = new Uint8Array(n);
    for (let i = 0; i < n; i++) out[i] = Module._web_peek((addr + i) & 0xffff);
    return out;
  };
  const refresh = () => {
    if (!available()) return;
    const ptr = Module._web_regs();
    const r = decodeRegs(Module.HEAPU8.subarray(ptr, ptr + 16));
    const near = nearestLabel(labels, r.pc);
    el("dbg-regs").textContent = `PC=${hex4(r.pc)}${near ? " (" + near.name + "+" + near.off + ")" : ""}  A=${r.a.toString(16).toUpperCase().padStart(2, "0")} X=${r.x.toString(16).toUpperCase().padStart(2, "0")} Y=${r.y.toString(16).toUpperCase().padStart(2, "0")} S=${r.s.toString(16).toUpperCase().padStart(2, "0")} P=${r.flags}  cycles=${r.cycles}`;
    const vars = el("dbg-vars");
    vars.innerHTML = "";
    for (const s of symbols) {
      const tr = document.createElement("tr");
      const v = s.kind === "str" ? JSON.stringify(decodeString(peek(s.addr, 256))) : String(decodeNumber(peek(s.addr, 5)));
      tr.innerHTML = "<td></td><td></td>";
      tr.children[0].textContent = s.name;
      tr.children[1].textContent = v;
      vars.appendChild(tr);
    }
    const addr = parseInt(el("dbg-addr").value, 16);
    if (!Number.isNaN(addr)) el("dbg-mem").textContent = hexDump(addr, peek(addr, 64));
  };
  const setPaused = (paused) => {
    el("dbg-state").textContent = paused ? "en pause" : "en cours";
    if (paused) { refresh(); if (!timer) timer = setInterval(refresh, 500); } else if (timer) { clearInterval(timer); timer = null; }
  };
  el("dbg-pause").addEventListener("click", () => { if (!available()) { status("Débogueur indisponible : reconstruire Phosphoneo (make wasm)", true); return; } Module._web_pause(); setPaused(true); });
  el("dbg-continue").addEventListener("click", () => { if (available()) { Module._web_resume(); setPaused(false); } });
  el("dbg-step").addEventListener("click", () => { if (available()) { Module._web_step(); setTimeout(refresh, 60); setPaused(true); } });
  el("dbg-refresh").addEventListener("click", refresh);
  el("dbg-addr").addEventListener("change", refresh);
  return {
    setSymbols(syms, labs) { symbols = syms || []; labels = labs || {}; if (el("dbg-addr").value === "" && symbols.length) el("dbg-addr").value = hex4(symbols[0].addr); refresh(); },
  };
}
