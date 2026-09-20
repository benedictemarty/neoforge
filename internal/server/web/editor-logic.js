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
