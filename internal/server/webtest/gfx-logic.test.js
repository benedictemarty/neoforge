import { test } from "node:test";
import assert from "node:assert/strict";
import { newObject, imageNumber, flipH, flipV, floodFill, setPixel, toSetJSON, fromSetJSON, canAdd, indexInKind, newTilemap, tilemapBytes, tilemapFromBytes, resizeTilemap, tilemapBasic, KINDS } from "../web/gfx-logic.js";

test("objets", () => {
  const t = newObject("tile16");
  assert.equal(t.pix.length, 256);
  assert.equal(t.pix[0], 8);
  assert.equal(newObject("sprite32").pix.length, 1024);
  assert.equal(newObject("sprite16").pix[0], 0);
  assert.equal(imageNumber("sprite16", 2), 0x82);
  assert.equal(imageNumber("sprite32", 0), 0xc0);
  let s = setPixel(newObject("sprite16"), 0, 1, 5);
  assert.equal(s.pix[16], 5);
  assert.equal(flipH(s).pix[16 + 15], 5);
  assert.equal(flipV(s).pix[14 * 16], 5);
  const f = floodFill(s, 5, 5, 3);
  assert.equal(f.pix[0], 3);
  assert.equal(f.pix[16], 5);
  assert.deepEqual(floodFill(s, 0, 1, 5).pix, s.pix);
  const objs = [newObject("tile16"), s, newObject("sprite16")];
  const j = toSetJSON(objs);
  assert.equal(j.tiles.length, 1);
  assert.equal(j.sprites16.length, 2);
  assert.equal(fromSetJSON(j).length, 3);
  assert.equal(fromSetJSON({}).length, 0);
  assert.equal(indexInKind(objs, 2), 1);
  assert.ok(canAdd(objs, "sprite32"));
  const many = Array.from({ length: KINDS.sprite32.max }, () => newObject("sprite32"));
  assert.ok(!canAdd(many, "sprite32"));
});

test("tilemap", () => {
  const m = newTilemap(3, 2);
  m.tiles[4] = 7;
  const b = tilemapBytes(m);
  assert.deepEqual(Array.from(b), [1, 3, 2, 0xf0, 0xf0, 0xf0, 0xf0, 7, 0xf0]);
  assert.deepEqual(tilemapFromBytes(b), m);
  assert.throws(() => tilemapFromBytes(new Uint8Array([2])));
  assert.throws(() => tilemapFromBytes(new Uint8Array([1, 4, 4, 0])));
  const r = resizeTilemap(m, 2, 3, 1);
  assert.deepEqual(r.tiles, [0xf0, 0xf0, 0xf0, 7, 1, 1]);
  const src = tilemapBasic(m);
  assert.match(src, /map = alloc\(9\)/);
  assert.match(src, /data 240,7,240/);
});
