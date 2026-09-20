# Changelog

Toutes les modifications notables de **neoforge** sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]
### Corrigé — exécution dans le navigateur (S1-2)
- `phosphoneo.js` résolvait `phosphoneo.data`/`.wasm` relativement à la page (404) : `Module.locateFile`
  les renvoie sous `/emu/`.
- Le programme par défaut contenait un « — » non ASCII, refusé par le tokeniseur.
### Ajouté
- Aide des commandes NeoBASIC (S1-5) : `help.json` généré par `tools/gen_help.py` depuis la
  documentation officielle **neo6502-documents** (MIT © Paul Robson, `reference/basic.md`) + extensions
  Trinity (`vmode`) — 190 entrées ; panneau ❔ (F1) filtrable avec insertion au clic, descriptions au
  survol et dans la complétion. Les mnémoniques 65C02 et les mots de structure (`endif`, `wend`…)
  n'ont pas d'entrée propre.
- Détokeniseur `.bas` → source (`internal/neobasic.List`, reproduction de `listbasic.py`, 61 listings
  du corpus identiques à la référence) ; `neobas -list [-n]`, `POST /api/detok`, bouton ⬆ .bas dans
  l'éditeur (S1-4). Limite héritée de la référence : `print .5` est listé `print.5`.
- `tools/browser_e2e.mjs` + `make e2e-browser` : Chrome headless piloté par CDP ouvre l'IDE, clique
  ▶ Exécuter et capture l'écran (programme vérifié en cours d'exécution dans Phosphoneo WASM).
- Dépôt distant public https://github.com/benedictemarty/neoforge et CI GitHub Actions
  (gofmt, vet, couverture 100 %, tests JS, build) — S1-1.

## [0.1.0] - 2026-09-20
### Ajouté — sprint 0, socle du portage de la Forge Oric sur Neo6502
- Cadrage (`docs/adr/ADR-001`) : modes `vmode 0`/`vmode 1` de Trinity, compilateur porté dès le début,
  sprites 16×16 + tilemaps, émulateur Phosphoneo en WebAssembly dans la page.
- `internal/neobasic` : table des tokens et tokeniseur NeoBASIC reproduits de `makebasic.py`
  (`#define`, `#library`, numérotation automatique) ; **75 programmes `.bsc` de Trinity identiques
  octet pour octet** à la référence (test différentiel, sauté sans `python3`).
- `cmd/neobas` : tokeniseur en ligne de commande (`-o`, `-library`, `-version`).
- `internal/server` + `web/` : page Monaco avec langage NeoBASIC (coloration, complétion, snippets
  sprite/tile/vmode, survol des tokens), Phosphoneo WASM servi sous `/emu/`, ▶ Exécuter (F5) =
  `/api/build` → `/storage/prog.bas` → `web_load_neo` ; Ouvrir/Enregistrer `.bsc` ; ⬇ `.bas`.
- `cmd/neoforge` : serveur (`NEOFORGE_ADDR`, `NEOFORGE_PHOSPHONEO_WEB`, `NEOFORGE_PROJECTS_DIR`).
- Qualité : couverture Go 100 % (`make cover-check`), tests JS (`node --test`), `make e2e`
  (Phosphoneo natif : `NEOFORGE OK`, `x=42` lus à l'écran), documentation agile (README,
  ARCHITECTURE, BACKLOG, ROADMAP, CHANGELOG).
