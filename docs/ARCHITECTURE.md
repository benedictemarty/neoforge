# Architecture — neoforge

## Vue d'ensemble

neoforge reprend l'architecture d'oriced (Forge Oric) en remplaçant deux maillons :

| Maillon            | oriced (Oric)                                  | neoforge (Neo6502)                                        |
|--------------------|------------------------------------------------|-----------------------------------------------------------|
| Source → binaire   | `bas2tap` (Phosphoric, C) → `.tap`             | `internal/neobasic` (Go, = `makebasic.py`) → `.bas`       |
| Émulateur          | Phosphoric natif headless + flux MJPEG + API   | **Phosphoneo WebAssembly dans la page** (canvas SDL)      |
| Chargement         | `--type-keys 'CLOAD…'` via API HTTP            | `Module.FS.writeFile` + `web_load_neo` (→ `load` + `run`) |
| Compilateur        | BASIC → 6502 sur les routines ROM Oric         | à porter : backend sur l'API firmware `$FF00` (épopée E5) |

```
┌──────────────── navigateur ────────────────────────────────────────────┐
│  Monaco (neobasic-lang.js)      │  <canvas> Phosphoneo (phosphoneo.js) │
│  app.js ── POST /api/build ──►  │  ◄── Module.FS + ccall(web_load_neo) │
└──────────────┬──────────────────┴──────────────────────────────────────┘
               │ HTTP/JSON                       ▲ GET /emu/* (statique)
┌──────────────▼──────────────────────────────────┴──────────────────────┐
│ neoforge (Go) : internal/server                                        │
│   /            page + vendor Monaco (go:embed)                         │
│   /emu/        NEOFORGE_PHOSPHONEO_WEB (phosphoneo.js/.wasm/.data)     │
│   /api/config  version, présence de l'émulateur                        │
│   /api/keywords table des tokens (nom, genre, ID)                      │
│   /api/build   source .bsc → .bas (internal/neobasic)                  │
│   /api/files, /api/file  programmes .bsc du répertoire de projets      │
└────────────────────────────────────────────────────────────────────────┘
```

## Paquets

- `internal/neobasic` — table des tokens NeoBASIC (reproduction de `tokens.py`, incl. la page
  d'extension Neo6502Basic `$380`/`$3C0`), tokeniseur (`tokeniser.py`), magasin d'identifiants,
  assemblage du programme (`makebasic.py` : `#define`, `#library`, numérotation automatique).
  Format `.bas` : `[pages × 256 o. d'identifiants][len][no lo][no hi][tokens…][$C0]…[0]`.
- `internal/config` — variables `NEOFORGE_*`.
- `internal/server` — gestionnaire HTTP ; `web/` embarqué (index.html, app.js, neobasic-lang.js,
  editor-logic.js testé sous node, style.css, vendor/vs = Monaco, help.json = aide générée par
  `tools/gen_help.py` depuis neo6502-documents).
- `cmd/neoforge` — serveur ; `cmd/neobas` — tokeniseur en ligne de commande.

## Émulateur

Phosphoneo est chargé comme un script emscripten classique (`Module` global, `--sdl --scale 1`).
Le `.bas` est écrit dans `/storage` (MEMFS) puis `web_load_neo` tape `load "nom.bas"` + `run` à
l'invite NeoBASIC (ou charge en page BASIC et réinitialise si le BASIC n'est pas à l'invite).
Les fichiers du build WASM ne sont **pas** versionnés ici : servis depuis `~/Phosphoneo/web`.

## Qualité

Couverture Go **100 %** (porte `make cover-check`), tests JS `node --test`, test différentiel
tokeniseur ↔ `makebasic.py` sur le corpus `.bsc` de Trinity, `make e2e` sur Phosphoneo natif.
