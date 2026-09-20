// Définition du langage NeoBASIC (Neo6502, firmware Trinity) pour Monaco :
// coloration syntaxique + autocomplétion. Les mots-clés viennent de /api/keywords.
import { keywordAt, splitKinds } from "/editor-logic.js";

export function registerNeoBasic(monaco, keywords) {
  const { statements, functions, structures, asm } = splitKinds(keywords);
  const kwByName = new Map(keywords.map((k) => [k.name, k]));

  monaco.languages.register({ id: "neobasic" });

  monaco.languages.setMonarchTokensProvider("neobasic", {
    ignoreCase: true,
    keywords: statements,
    functions,
    structures,
    asm,
    tokenizer: {
      root: [
        [/^\s*\d+/, "number.line"],
        [/^\s*#\w+.*$/, "keyword.directive"],
        [/\/\/.*$/, "comment"],
        [/'.*$/, "comment"],
        [/"[^"]*"?/, "string"],
        [/\$[0-9A-Fa-f]+/, "number.hex"],
        [/\d+(\.\d+)?/, "number"],
        [/[A-Za-z][A-Za-z0-9_.]*\$?\(?/, {
          cases: {
            "@structures": "keyword.control",
            "@keywords": "keyword",
            "@functions": "type.identifier",
            "@asm": "keyword.asm",
            "@default": "identifier",
          },
        }],
        [/[-+*/^=<>:;,()\[\]@#%&|\\]/, "delimiter"],
      ],
    },
  });

  monaco.languages.setLanguageConfiguration("neobasic", {
    comments: { lineComment: "'" },
    brackets: [["(", ")"], ["[", "]"]],
    autoClosingPairs: [
      { open: "(", close: ")" },
      { open: "[", close: "]" },
      { open: '"', close: '"' },
    ],
  });

  const SNIPPETS = [
    { label: "for", detail: "Boucle FOR … NEXT", body: "for ${1:i} = ${2:1} to ${3:10}\n\t$0\nnext" },
    { label: "while", detail: "Boucle WHILE … WEND", body: "while ${1:cond}\n\t$0\nwend" },
    { label: "repeat", detail: "Boucle REPEAT … UNTIL", body: "repeat\n\t$0\nuntil ${1:cond}" },
    { label: "ifendif", detail: "IF … ENDIF (bloc)", body: "if ${1:cond}\n\t$0\nendif" },
    { label: "proc", detail: "Procédure PROC … ENDPROC", body: "proc ${1:nom}(${2})\n\t$0\nendproc" },
    { label: "sprite", detail: "Sprite : image, position, dessin", body: "sprite ${1:0} image ${2:0} to ${3:100},${4:100}$0" },
    { label: "tile", detail: "Tilemap : définition + affichage", body: "tilemap ${1:addr},${2:w},${3:h}\ntiledraw ${4:0},${5:0} to ${6:320},${7:240}$0" },
    { label: "vmode", detail: "Mode vidéo (0 = 320×240, 1 = Hercules 80 col.)", body: "vmode ${1:0}$0" },
  ];

  monaco.languages.registerCompletionItemProvider("neobasic", {
    provideCompletionItems(model, position) {
      const word = model.getWordUntilPosition(position);
      const range = { startLineNumber: position.lineNumber, endLineNumber: position.lineNumber, startColumn: word.startColumn, endColumn: word.endColumn };
      const kind = (k) => (k === "function" ? monaco.languages.CompletionItemKind.Function : k === "asm" ? monaco.languages.CompletionItemKind.Operator : monaco.languages.CompletionItemKind.Keyword);
      const items = keywords.map((k) => ({ label: k.name.toLowerCase(), kind: kind(k.kind), detail: k.kind, insertText: k.name.toLowerCase(), range }));
      for (const s of SNIPPETS) {
        items.push({ label: s.label, kind: monaco.languages.CompletionItemKind.Snippet, detail: s.detail, insertText: s.body, insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet, range });
      }
      return { suggestions: items };
    },
  });

  monaco.languages.registerHoverProvider("neobasic", {
    provideHover(model, position) {
      const k = keywordAt(model.getLineContent(position.lineNumber), position.column - 1, kwByName);
      return k ? { contents: [{ value: "**" + k.name + "** — " + k.kind + " (token $" + k.id.toString(16).toUpperCase() + ")" }] } : null;
    },
  });
}
