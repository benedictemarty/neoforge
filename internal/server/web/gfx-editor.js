// Éditeur graphique (sprites 16×16 / 32×32, tuiles 16×16, tilemap) — interface DOM.
// La logique pure est dans gfx-logic.js ; les conversions .gfx/PNG passent par /api/gfx.
import { PALETTE, KINDS, newObject, imageNumber, flipH, flipV, floodFill, setPixel, toSetJSON, fromSetJSON, canAdd, indexInKind, newTilemap, tilemapBytes, tilemapFromBytes, resizeTilemap, tilemapBasic } from "/gfx-logic.js";
import { decodeBase64 } from "/editor-logic.js";

const el = (id) => document.getElementById(id);
const rgb = (c) => `rgb(${PALETTE[c][0]},${PALETTE[c][1]},${PALETTE[c][2]})`;

export function setupGfxEditor({ status, sendToStorage, insertText }) {
  let objects = [];
  let current = -1;
  let colour = 7;
  let tool = "pen"; // pen | fill
  let map = newTilemap(20, 15);
  let mapTile = 0;
  const ZOOM = 16;

  const listEl = el("gfx-list"), canvas = el("gfx-canvas"), ctx = canvas.getContext("2d");
  const mapCanvas = el("map-canvas"), mctx = mapCanvas.getContext("2d");

  // ─── Rendu ──────────────────────────────────────────────────────────────
  const drawObject = (o, c, scale, ox = 0, oy = 0) => {
    const n = KINDS[o.kind].size;
    for (let y = 0; y < n; y++) for (let x = 0; x < n; x++) {
      const p = o.pix[y * n + x];
      if (p === 0 && o.kind !== "tile16") { c.fillStyle = (x + y) & 1 ? "#333" : "#444"; } else c.fillStyle = rgb(p);
      c.fillRect(ox + x * scale, oy + y * scale, scale, scale);
    }
  };
  const renderList = () => {
    listEl.innerHTML = "";
    objects.forEach((o, i) => {
      const d = document.createElement("div");
      d.className = "gfx-item" + (i === current ? " active" : "");
      const th = document.createElement("canvas");
      th.width = th.height = 32;
      drawObject(o, th.getContext("2d"), 32 / KINDS[o.kind].size);
      const label = document.createElement("span");
      label.textContent = "$" + imageNumber(o.kind, indexInKind(objects, i)).toString(16).toUpperCase().padStart(2, "0");
      d.append(th, label);
      d.addEventListener("click", () => select(i));
      listEl.appendChild(d);
    });
    el("gfx-count").textContent = objects.length + " objet(s)";
  };
  const renderCanvas = () => {
    if (current < 0) { canvas.width = canvas.height = 16 * ZOOM; ctx.clearRect(0, 0, canvas.width, canvas.height); return; }
    const o = objects[current], n = KINDS[o.kind].size;
    canvas.width = canvas.height = n * ZOOM;
    drawObject(o, ctx, ZOOM);
    ctx.strokeStyle = "rgba(255,255,255,.15)";
    for (let i = 0; i <= n; i++) { ctx.beginPath(); ctx.moveTo(i * ZOOM, 0); ctx.lineTo(i * ZOOM, n * ZOOM); ctx.moveTo(0, i * ZOOM); ctx.lineTo(n * ZOOM, i * ZOOM); ctx.stroke(); }
  };
  const renderPalette = () => {
    const p = el("gfx-palette");
    p.innerHTML = "";
    for (let c = 0; c < 16; c++) {
      const s = document.createElement("button");
      s.className = "swatch" + (c === colour ? " active" : "");
      s.style.background = c === 0 ? "repeating-linear-gradient(45deg,#333,#333 4px,#555 4px,#555 8px)" : rgb(c);
      s.title = c === 0 ? "0 : transparent (sprites)" : "couleur " + c;
      s.addEventListener("click", () => { colour = c; renderPalette(); });
      p.appendChild(s);
    }
  };
  const tiles = () => objects.filter((o) => o.kind === "tile16");
  const renderMap = () => {
    const T = 16;
    mapCanvas.width = map.w * T; mapCanvas.height = map.h * T;
    const ts = tiles();
    for (let y = 0; y < map.h; y++) for (let x = 0; x < map.w; x++) {
      const t = map.tiles[y * map.w + x];
      if (t < ts.length) drawObject(ts[t], mctx, 1, x * T, y * T);
      else { mctx.fillStyle = t === 0xf0 ? "#222" : t >= 0xf1 ? rgb(t - 0xf0) : "#800"; mctx.fillRect(x * T, y * T, T, T); }
    }
    el("map-size").textContent = map.w + "×" + map.h;
    el("map-tile").textContent = "tuile $" + mapTile.toString(16).toUpperCase().padStart(2, "0");
  };
  const select = (i) => { current = i; renderList(); renderCanvas(); };
  const update = (o) => { objects[current] = o; renderCanvas(); renderList(); renderMap(); };

  // ─── Édition de pixels ──────────────────────────────────────────────────
  const pixelAt = (ev) => {
    const r = canvas.getBoundingClientRect();
    const n = KINDS[objects[current].kind].size;
    const x = Math.floor((ev.clientX - r.left) / r.width * n), y = Math.floor((ev.clientY - r.top) / r.height * n);
    return x >= 0 && y >= 0 && x < n && y < n ? [x, y] : null;
  };
  let painting = false;
  const paint = (ev) => {
    if (current < 0) return;
    const p = pixelAt(ev);
    if (!p) return;
    update(tool === "fill" ? floodFill(objects[current], p[0], p[1], colour) : setPixel(objects[current], p[0], p[1], colour));
  };
  canvas.addEventListener("mousedown", (ev) => { painting = tool === "pen"; paint(ev); });
  canvas.addEventListener("mousemove", (ev) => { if (painting) paint(ev); });
  window.addEventListener("mouseup", () => { painting = false; });
  el("gfx-pen").addEventListener("click", () => { tool = "pen"; });
  el("gfx-fill").addEventListener("click", () => { tool = "fill"; });
  el("gfx-fliph").addEventListener("click", () => { if (current >= 0) update(flipH(objects[current])); });
  el("gfx-flipv").addEventListener("click", () => { if (current >= 0) update(flipV(objects[current])); });
  el("gfx-clear").addEventListener("click", () => { if (current >= 0) update(newObject(objects[current].kind)); });
  el("gfx-delete").addEventListener("click", () => { if (current >= 0) { objects.splice(current, 1); current = Math.min(current, objects.length - 1); renderList(); renderCanvas(); renderMap(); } });
  for (const kind of ["tile16", "sprite16", "sprite32"]) {
    el("gfx-add-" + kind).addEventListener("click", () => {
      if (!canAdd(objects, kind)) { status("Maximum atteint pour " + kind, true); return; }
      objects.push(newObject(kind));
      objects.sort((a, b) => Object.keys(KINDS).indexOf(a.kind) - Object.keys(KINDS).indexOf(b.kind)); // ordre du fichier .gfx
      select(objects.lastIndexOf(objects.find((o) => o.kind === kind && o === objects[objects.length - 1]) || objects[objects.length - 1]));
    });
  }

  // ─── Fichiers ───────────────────────────────────────────────────────────
  const download = (name, bytes, type = "application/octet-stream") => {
    const a = document.createElement("a");
    a.href = URL.createObjectURL(new Blob([bytes], { type }));
    a.download = name;
    a.click();
  };
  const renderGfx = async () => {
    const j = await fetch("/api/gfx/render", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(toSetJSON(objects)) }).then((r) => r.json());
    if (j.error) { status(j.error, true); return null; }
    return decodeBase64(j.gfx);
  };
  el("gfx-open").addEventListener("change", async (ev) => {
    const f = ev.target.files[0];
    if (!f) return;
    const j = await fetch("/api/gfx/parse", { method: "POST", body: await f.arrayBuffer() }).then((r) => r.json());
    if (j.error) { status(j.error, true); return; }
    objects = fromSetJSON(j);
    select(objects.length ? 0 : -1);
    renderMap();
    status(f.name + " : " + objects.length + " objet(s)");
    ev.target.value = "";
  });
  el("gfx-save").addEventListener("click", async () => { const b = await renderGfx(); if (b) download(el("gfx-name").value || "graphics.gfx", b); });
  el("gfx-storage").addEventListener("click", async () => { const b = await renderGfx(); if (b) sendToStorage(el("gfx-name").value || "graphics.gfx", b); });
  el("gfx-sheet-in").addEventListener("change", async (ev) => {
    const f = ev.target.files[0];
    if (!f) return;
    const kind = el("gfx-sheet-kind").value;
    const j = await fetch("/api/gfx/sheet/import?kind=" + kind, { method: "POST", body: await f.arrayBuffer() }).then((r) => r.json());
    if (j.error) { status(j.error, true); return; }
    for (const p of j.objects) if (canAdd(objects, kind)) objects.push({ kind, pix: p });
    objects.sort((a, b) => Object.keys(KINDS).indexOf(a.kind) - Object.keys(KINDS).indexOf(b.kind));
    select(objects.length ? 0 : -1);
    renderMap();
    status(f.name + " : " + j.objects.length + " objet(s) importé(s)");
    ev.target.value = "";
  });
  el("gfx-sheet-out").addEventListener("click", async () => {
    const kind = el("gfx-sheet-kind").value;
    const objs = objects.filter((o) => o.kind === kind).map((o) => Array.from(o.pix));
    const r = await fetch("/api/gfx/sheet/export?kind=" + kind, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ objects: objs }) });
    if (!r.ok) { status("export impossible", true); return; }
    download({ tile16: "tile_16.png", sprite16: "sprite_16.png", sprite32: "sprite_32.png" }[kind], await r.arrayBuffer(), "image/png");
  });

  el("gfx-image-in").addEventListener("change", async (ev) => {
    const f = ev.target.files[0];
    if (!f) return;
    const j = await fetch("/api/gfx/image", { method: "POST", body: await f.arrayBuffer() }).then((r) => r.json());
    if (j.error) { status(j.error, true); return; }
    objects = objects.filter((o) => o.kind !== "tile16").concat(j.tiles.map((p) => ({ kind: "tile16", pix: p })));
    objects.sort((a, b) => Object.keys(KINDS).indexOf(a.kind) - Object.keys(KINDS).indexOf(b.kind));
    map = { w: j.map.w, h: j.map.h, tiles: j.map.tiles };
    el("map-w").value = map.w; el("map-h").value = map.h;
    select(objects.length ? 0 : -1);
    renderMap();
    status(f.name + " : " + j.tiles.length + " tuile(s), carte " + map.w + "×" + map.h + " (tuiles précédentes remplacées)");
    ev.target.value = "";
  });

  // ─── Tilemap ────────────────────────────────────────────────────────────
  const mapCell = (ev) => {
    const r = mapCanvas.getBoundingClientRect();
    return [Math.floor((ev.clientX - r.left) / r.width * map.w), Math.floor((ev.clientY - r.top) / r.height * map.h)];
  };
  let mapPainting = false;
  const mapPaint = (ev) => { const [x, y] = mapCell(ev); if (x >= 0 && y >= 0 && x < map.w && y < map.h) { map.tiles[y * map.w + x] = mapTile; renderMap(); } };
  mapCanvas.addEventListener("mousedown", (ev) => { mapPainting = true; mapPaint(ev); });
  mapCanvas.addEventListener("mousemove", (ev) => { if (mapPainting) mapPaint(ev); });
  window.addEventListener("mouseup", () => { mapPainting = false; });
  mapCanvas.addEventListener("contextmenu", (ev) => { ev.preventDefault(); const [x, y] = mapCell(ev); mapTile = map.tiles[y * map.w + x]; renderMap(); });
  el("map-use").addEventListener("click", () => { if (current >= 0 && objects[current].kind === "tile16") { mapTile = indexInKind(objects, current); renderMap(); } });
  el("map-transparent").addEventListener("click", () => { mapTile = 0xf0; renderMap(); });
  el("map-solid").addEventListener("click", () => { mapTile = 0xf0 + colour; renderMap(); });
  el("map-resize").addEventListener("click", () => {
    const w = parseInt(el("map-w").value, 10), h = parseInt(el("map-h").value, 10);
    if (w >= 1 && h >= 1 && w <= 255 && h <= 255) { map = resizeTilemap(map, w, h); renderMap(); }
  });
  el("map-save").addEventListener("click", () => download(el("map-name").value || "level.map", tilemapBytes(map)));
  el("map-storage").addEventListener("click", () => sendToStorage(el("map-name").value || "level.map", tilemapBytes(map)));
  el("map-basic").addEventListener("click", () => insertText(tilemapBasic(map)));
  el("map-open").addEventListener("change", async (ev) => {
    const f = ev.target.files[0];
    if (!f) return;
    try { map = tilemapFromBytes(new Uint8Array(await f.arrayBuffer())); el("map-w").value = map.w; el("map-h").value = map.h; renderMap(); status(f.name + " : " + map.w + "×" + map.h); }
    catch (e) { status(String(e.message || e), true); }
    ev.target.value = "";
  });

  renderPalette();
  renderList();
  renderCanvas();
  renderMap();
  return { objects: () => objects };
}
