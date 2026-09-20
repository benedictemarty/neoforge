// Logique pure de l'éditeur graphique (sans DOM), testée sous node.
// Un objet = { kind: "tile16" | "sprite16" | "sprite32", pix: Uint8Array/Array de size*size indices 0..15 }.

export const PALETTE = [
  [0, 0, 0], [255, 0, 77], [0, 228, 54], [255, 236, 39], [29, 43, 83], [126, 37, 83], [41, 173, 255], [255, 241, 232],
  [0, 0, 0], [95, 87, 79], [0, 135, 81], [255, 163, 0], [171, 82, 54], [131, 118, 156], [255, 204, 170], [194, 195, 199],
];

export const KINDS = { tile16: { size: 16, base: 0x00, max: 128 }, sprite16: { size: 16, base: 0x80, max: 64 }, sprite32: { size: 32, base: 0xc0, max: 64 } };

// newObject : objet vide (tuile : couleur 8 = noir opaque ; sprite : 0 = transparent).
export function newObject(kind) {
  const n = KINDS[kind].size;
  return { kind, pix: new Array(n * n).fill(kind === "tile16" ? 8 : 0) };
}

// imageNumber : numéro d'image d'un objet (tuiles $00…, sprites 16 $80…, sprites 32 $C0…).
export function imageNumber(kind, index) {
  return KINDS[kind].base + index;
}

// flipH / flipV : retournement horizontal / vertical (nouvel objet).
export function flipH(o) {
  const n = KINDS[o.kind].size;
  const pix = o.pix.slice();
  for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) pix[y * n + x] = o.pix[y * n + (n - 1 - x)];
  return { kind: o.kind, pix };
}

export function flipV(o) {
  const n = KINDS[o.kind].size;
  const pix = o.pix.slice();
  for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) pix[y * n + x] = o.pix[(n - 1 - y) * n + x];
  return { kind: o.kind, pix };
}

// floodFill : remplissage (4-connexe) de la zone de même couleur à partir de (x, y).
export function floodFill(o, x, y, colour) {
  const n = KINDS[o.kind].size;
  const pix = o.pix.slice();
  const from = pix[y * n + x];
  if (from === colour) return { kind: o.kind, pix };
  const stack = [[x, y]];
  while (stack.length) {
    const [cx, cy] = stack.pop();
    if (cx < 0 || cy < 0 || cx >= n || cy >= n || pix[cy * n + cx] !== from) continue;
    pix[cy * n + cx] = colour;
    stack.push([cx + 1, cy], [cx - 1, cy], [cx, cy + 1], [cx, cy - 1]);
  }
  return { kind: o.kind, pix };
}

// setPixel : nouvel objet avec le pixel (x, y) à colour.
export function setPixel(o, x, y, colour) {
  const n = KINDS[o.kind].size;
  const pix = o.pix.slice();
  pix[y * n + x] = colour & 15;
  return { kind: o.kind, pix };
}

// toSetJSON / fromSetJSON : forme échangée avec /api/gfx (tableaux par genre).
export function toSetJSON(objects) {
  const j = { tiles: [], sprites16: [], sprites32: [] };
  for (const o of objects) j[{ tile16: "tiles", sprite16: "sprites16", sprite32: "sprites32" }[o.kind]].push(Array.from(o.pix));
  return j;
}

export function fromSetJSON(j) {
  const out = [];
  for (const p of j.tiles || []) out.push({ kind: "tile16", pix: p });
  for (const p of j.sprites16 || []) out.push({ kind: "sprite16", pix: p });
  for (const p of j.sprites32 || []) out.push({ kind: "sprite32", pix: p });
  return out;
}

// canAdd : respecte les maxima par genre (128 tuiles, 64 sprites de chaque taille).
export function canAdd(objects, kind) {
  return objects.filter((o) => o.kind === kind).length < KINDS[kind].max;
}

// indexInKind : index de l'objet parmi ceux de son genre (= numéro d'image − base).
export function indexInKind(objects, i) {
  return objects.slice(0, i).filter((o) => o.kind === objects[i].kind).length;
}

// ─── Tilemap ────────────────────────────────────────────────────────────────

export function newTilemap(w, h, fill = 0xf0) {
  return { w, h, tiles: new Array(w * h).fill(fill) };
}

// tilemapBytes : format mémoire [1][w][h][tuiles…] (chargeable par load "f",adr).
export function tilemapBytes(m) {
  return Uint8Array.from([1, m.w, m.h, ...m.tiles]);
}

export function tilemapFromBytes(b) {
  if (b.length < 3 || b[0] !== 1) throw new Error("tilemap invalide (en-tête)");
  const w = b[1], h = b[2];
  if (w === 0 || h === 0 || b.length < 3 + w * h) throw new Error("tilemap tronquée");
  return { w, h, tiles: Array.from(b.slice(3, 3 + w * h)) };
}

// resizeTilemap : nouvelle taille, contenu conservé en haut à gauche.
export function resizeTilemap(m, w, h, fill = 0xf0) {
  const out = newTilemap(w, h, fill);
  for (let y = 0; y < Math.min(h, m.h); y++) for (let x = 0; x < Math.min(w, m.w); x++) out.tiles[y * w + x] = m.tiles[y * m.w + x];
  return out;
}

// tilemapBasic : lignes NeoBASIC qui reconstruisent la carte en mémoire (alloc + poke),
// pour les programmes sans fichier .map.
export function tilemapBasic(m, varName = "map") {
  const lines = [`${varName} = alloc(${m.w * m.h + 3})`, `poke ${varName},1: poke ${varName}+1,${m.w}: poke ${varName}+2,${m.h}`];
  const rows = [];
  for (let y = 0; y < m.h; y++) rows.push(m.tiles.slice(y * m.w, (y + 1) * m.w).join(","));
  lines.push(`for i = 0 to ${m.w * m.h - 1}: read t: poke ${varName}+3+i,t: next`);
  for (const r of rows) lines.push("data " + r);
  return lines.join("\n") + "\n";
}
