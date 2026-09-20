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
  assert.equal(storageName("mon jeu.gfx", true), "mon_jeu.gfx");
  assert.equal(storageName("", true), "data");
});

test("splitKinds", () => {
  const r = splitKinds([{ name: "PRINT", kind: "statement" }, { name: "RND(", kind: "function" }, { name: "IF", kind: "structure" }, { name: "LDA", kind: "asm" }]);
  assert.deepEqual(r, { statements: ["PRINT"], functions: ["RND("], structures: ["IF"], asm: ["LDA"] });
});

import { filterHelp, helpByName } from "../web/editor-logic.js";

test("filterHelp / helpByName", () => {
  const entries = [
    { name: "print", syntax: "print …", notes: "Affiche", section: "s" },
    { name: "rnd(", syntax: "rnd(n)", notes: "Random number", section: "s" },
  ];
  assert.equal(filterHelp(entries, "").length, 2);
  assert.deepEqual(filterHelp(entries, "RANDOM").map((e) => e.name), ["rnd("]);
  assert.deepEqual(filterHelp(entries, "print").map((e) => e.name), ["print"]);
  assert.equal(helpByName(entries).get("RND(").syntax, "rnd(n)");
});

import { tabsAdd, tabsActivate, tabsClose, tabsFindByName, tabTitle } from "../web/editor-logic.js";

test("onglets", () => {
  let s = { tabs: [], activeId: null };
  s = tabsAdd(s, { id: 1, name: "a.bsc", savedVersion: 1 });
  s = tabsAdd(s, { id: 2, name: "", savedVersion: 1 });
  s = tabsAdd(s, { id: 3, name: "c.bsc", savedVersion: 1 });
  assert.equal(s.activeId, 3);
  assert.equal(tabsActivate(s, 1).activeId, 1);
  assert.equal(tabsActivate(s, 99), s);
  assert.equal(tabsFindByName(s, "A.BSC").id, 1);
  assert.equal(tabsFindByName(s, "zz"), undefined);
  s = tabsClose(s, 3);
  assert.deepEqual(s.tabs.map((t) => t.id), [1, 2]);
  assert.equal(s.activeId, 2);
  s = tabsClose(s, 1);
  assert.equal(s.activeId, 2);
  assert.equal(tabsClose(s, 42), s);
  s = tabsClose(s, 2);
  assert.equal(s.activeId, null);
  assert.equal(tabTitle({ name: "", savedVersion: 1 }, 1), "sans titre");
  assert.equal(tabTitle({ name: "a.bsc", savedVersion: 1 }, 2), "a.bsc ●");
});
