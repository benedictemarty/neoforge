// Logique pure de l'éditeur (sans Monaco ni DOM), testée par `node --test`
// (../webtest/). app.js et neobasic-lang.js importent ces fonctions.

// keywordAt : sous col (0-based) de line, renvoie le mot-clé (Map kwByName, clés
// en majuscules) couvrant le curseur ; gère les suffixes « $ » et « ( » collés
// (LEFT$(, RND(). Renvoie l'objet Keyword ou null.
export function keywordAt(line, col, kwByName) {
  const re = /[A-Za-z][A-Za-z0-9_.]*\$?\(?/g;
  let m;
  while ((m = re.exec(line))) {
    if (col < m.index || col > m.index + m[0].length) continue;
    const cand = m[0].toUpperCase();
    return kwByName.get(cand) || (cand.endsWith("(") && kwByName.get(cand.slice(0, -1))) || null;
  }
  return null;
}

// decodeBase64 : chaîne base64 (encodage Go de []byte) → Uint8Array.
export function decodeBase64(b64) {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

// errorLine : extrait le numéro de ligne (1-based) d'un message « ligne N : … »
// du tokeniseur, ou 0.
export function errorLine(msg) {
  const m = /^ligne (\d+) :/.exec(msg || "");
  return m ? parseInt(m[1], 10) : 0;
}

// storageName : nom de fichier sûr pour /storage de l'émulateur (comme la page
// Phosphoneo). Par défaut le nom d'un programme : « .bsc » → « .bas », vide → prog.bas ;
// keep = true conserve l'extension (fichiers de données).
export function storageName(name, keep = false) {
  const safe = (name || "").replace(/[^A-Za-z0-9_.-]/g, "_");
  if (keep) return safe || "data";
  return (safe.replace(/\.bsc$/i, "") || "prog") + ".bas";
}

// splitKinds : répartit les mots-clés par genre pour la coloration Monaco.
export function splitKinds(keywords) {
  const by = (k) => keywords.filter((x) => x.kind === k).map((x) => x.name);
  return { statements: by("statement"), functions: by("function"), structures: by("structure"), asm: by("asm") };
}

// filterHelp : entrées d'aide dont le nom, la syntaxe ou la description contient
// le filtre (insensible à la casse) ; filtre vide = toutes, triées par nom.
export function filterHelp(entries, filter) {
  const f = (filter || "").trim().toLowerCase();
  return entries.filter((e) => !f || e.name.includes(f) || e.syntax.toLowerCase().includes(f) || e.notes.toLowerCase().includes(f));
}

// helpByName : Map nom de token (majuscules) → entrée d'aide, pour le survol.
export function helpByName(entries) {
  return new Map(entries.map((e) => [e.name.toUpperCase(), e]));
}

// ─── Onglets (S1-3) : état pur, sans DOM ni Monaco ───────────────────────────
// Un onglet = { id, name, savedVersion } ; le modèle Monaco est tenu par app.js.
// Les fonctions renvoient un nouvel état { tabs, activeId }.
export function tabsAdd(state, tab) {
  return { tabs: [...state.tabs, tab], activeId: tab.id };
}

export function tabsActivate(state, id) {
  return state.tabs.some((t) => t.id === id) ? { ...state, activeId: id } : state;
}

// tabsClose : ferme l'onglet ; l'actif devient le voisin de gauche (ou de droite).
export function tabsClose(state, id) {
  const i = state.tabs.findIndex((t) => t.id === id);
  if (i < 0) return state;
  const tabs = state.tabs.filter((t) => t.id !== id);
  let activeId = state.activeId;
  if (activeId === id) activeId = tabs.length ? tabs[Math.max(0, i - 1)].id : null;
  return { tabs, activeId };
}

// tabsFindByName : onglet portant ce nom de fichier (insensible à la casse), ou undefined.
export function tabsFindByName(state, name) {
  return state.tabs.find((t) => t.name && t.name.toLowerCase() === (name || "").toLowerCase());
}

// tabTitle : libellé « nom ● » quand la version courante diffère de la version enregistrée.
export function tabTitle(tab, currentVersion) {
  return (tab.name || "sans titre") + (currentVersion !== tab.savedVersion ? " ●" : "");
}

// ─── Débogueur (S5-4) : décodages purs ─────────────────────────────────────

// decodeRegs : 16 octets de web_regs → { pc, a, x, y, s, p, cycles, flags }.
export function decodeRegs(b) {
  let cycles = 0;
  for (let i = 7; i >= 0; i--) cycles = cycles * 256 + b[8 + i];
  const p = b[6];
  const flags = ["N", "V", "-", "B", "D", "I", "Z", "C"].map((f, i) => ((p >> (7 - i)) & 1 ? f : f.toLowerCase())).join("");
  return { pc: b[0] | (b[1] << 8), a: b[2], x: b[3], y: b[4], s: b[5], p, cycles, flags };
}

// decodeNumber : 5 octets [type][valeur] → nombre (entier 32 bits signé ou float32).
export function decodeNumber(b) {
  const dv = new DataView(new ArrayBuffer(4));
  for (let i = 0; i < 4; i++) dv.setUint8(i, b[1 + i]);
  return b[0] & 0x40 ? dv.getFloat32(0, true) : dv.getInt32(0, true);
}

// decodeString : [longueur][caractères] → chaîne.
export function decodeString(b) {
  let s = "";
  for (let i = 0; i < b[0]; i++) s += String.fromCharCode(b[1 + i]);
  return s;
}

// hexDump : octets à partir de addr, 16 par ligne, hexadécimal + ASCII.
export function hexDump(addr, bytes) {
  const lines = [];
  for (let i = 0; i < bytes.length; i += 16) {
    const row = Array.from(bytes.slice(i, i + 16));
    const hex = row.map((v) => v.toString(16).toUpperCase().padStart(2, "0")).join(" ");
    const asc = row.map((v) => (v >= 32 && v < 127 ? String.fromCharCode(v) : ".")).join("");
    lines.push((addr + i).toString(16).toUpperCase().padStart(4, "0") + "  " + hex.padEnd(47) + "  " + asc);
  }
  return lines.join("\n");
}

// hex4 : nombre → 4 chiffres hexadécimaux.
export function hex4(n) {
  return (n & 0xffff).toString(16).toUpperCase().padStart(4, "0");
}

// nearestLabel : étiquette (nom, décalage) la plus proche en dessous de addr, ou null. Les
// étiquettes locales du générateur (minuscules suivies de « _numéro » : else_12, forc_3, apiw_7…)
// sont ignorées : seules restent les repères utiles (PROC_…, RT_…, VAR_…, STR_…, STK, ENDPROG…).
export function nearestLabel(labels, addr) {
  let best = null;
  for (const [name, a] of Object.entries(labels || {})) {
    if (a <= addr && (!best || a > best.addr) && !/^[a-z][a-z0-9]*_\d+$/.test(name)) best = { name, addr: a };
  }
  return best ? { name: best.name, off: addr - best.addr } : null;
}

// decodeRuntimeError : 3 octets d'ERRINFO ([code][ligne lo][ligne hi]) → null si aucune erreur,
// sinon { message, basLine, srcLine } (srcLine = 0 si la ligne BASIC est inconnue).
export function decodeRuntimeError(bytes, messages, lines) {
  if (!bytes || !bytes[0]) return null;
  const basLine = bytes[1] | (bytes[2] << 8);
  const message = (messages || [])[bytes[0] - 1] || "Erreur d'exécution";
  return { message, basLine, srcLine: (lines || {})[basLine] || 0 };
}

// ─── Carte réelle (S6-3) ────────────────────────────────────────────────────

// receiverProgram : programme NeoBASIC qui télécharge `name` depuis neoforge (adresse
// hôte:port) par tranches de 200 octets via atget$( et l'enregistre sur la clé du Neo6502.
// wifi = { ssid, pwd } ajoute la connexion Wi-Fi (atconnect) en tête.
export function receiverProgram(addr, name, size, wifi) {
  const q = '"';
  const lines = ["' neoforge : reception de " + name + " (" + size + " octets)"];
  if (wifi && wifi.ssid) lines.push("atconnect " + q + wifi.ssid + q + ", " + q + (wifi.pwd || "") + q);
  lines.push(
    "u$ = " + q + "http://" + addr + "/api/xfer/" + name + q,
    "n = val(atget$(u$ + " + q + "/size" + q + "))",
    "if n = 0 then print " + q + "serveur injoignable : " + q + "; atresult$: end",
    "base = alloc(n + 1)",
    "i = 0",
    "while i * 200 < n",
    "  t$ = atget$(u$ + " + q + "?c=" + q + " + str$(i))",
    "  for j = 1 to len(t$): poke base + i * 200 + j - 1, asc(mid$(t$, j, 1)): next",
    "  print " + q + "." + q + ";",
    "  i = i + 1",
    "wend",
    "save " + q + name + q + ", base, n",
    "print: print " + q + name + " recu (" + q + "; n; " + q + " octets)" + q,
  );
  return lines.join("\n") + "\n";
}
