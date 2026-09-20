import { test } from "node:test";
import assert from "node:assert/strict";
import { keywordAt, decodeBase64, errorLine, storageName, splitKinds } from "../web/editor-logic.js";

const kw = new Map([["PRINT", { name: "PRINT" }], ["LEFT$(", { name: "LEFT$(" }], ["RND(", { name: "RND(" }], ["TRUE", { name: "TRUE" }]]);

test("keywordAt", () => {
  assert.equal(keywordAt('print left$(a$,2)', 1, kw).name, "PRINT");
  assert.equal(keywordAt('print left$(a$,2)', 8, kw).name, "LEFT$(");
  assert.equal(keywordAt('x = true', 6, kw).name, "TRUE");
  assert.equal(keywordAt('x = rnd(', 5, kw).name, "RND(");
  assert.equal(keywordAt('x = zorg', 5, kw), null);
  assert.equal(keywordAt('x = 12', 5, kw), null);
});

test("decodeBase64", () => {
  assert.deepEqual(Array.from(decodeBase64("AQID")), [1, 2, 3]);
});

test("errorLine", () => {
  assert.equal(errorLine("ligne 12 : chaîne non terminée"), 12);
  assert.equal(errorLine("autre"), 0);
  assert.equal(errorLine(undefined), 0);
});

test("storageName", () => {
  assert.equal(storageName("mon jeu.bsc"), "mon_jeu.bas");
  assert.equal(storageName(""), "prog.bas");
});

test("splitKinds", () => {
  const r = splitKinds([{ name: "PRINT", kind: "statement" }, { name: "RND(", kind: "function" }, { name: "IF", kind: "structure" }, { name: "LDA", kind: "asm" }]);
  assert.deepEqual(r, { statements: ["PRINT"], functions: ["RND("], structures: ["IF"], asm: ["LDA"] });
});
