# Changelog

Toutes les modifications notables de **neoforge** sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]
### Ajouté — compilateur : entrées et chaînes (S4-1)
- `input` (boucle de saisie identique à l'interpréteur : 80 caractères, écho, retour arrière, « ?? » si le
  nombre est invalide), `inkey$(`, comparaisons de chaînes, `left$( right$( mid$( instr( val( isval( upper$(
  lower$( spc(` ; harnais différentiel avec frappe automatique (`testdata/<nom>.keys`) — 9 programmes identiques.

## [0.4.0] - 2026-09-21
### Ajouté — sprint 3, sprites, tuiles, tilemaps
- `internal/gfx` : modèle de `graphics.gfx` (4 bpp, palette du firmware), planches PNG au format `makeimg`
  (import **identique à la sortie du script officiel** sur les planches crossdev, export réimportable),
  tilemaps `[1][w][h][…]`, import d'image → tuiles dédupliquées + carte (S3-1, S3-3).
- API `/api/gfx/{parse,render,sheet/import,sheet/export,image}`.
- Éditeur **🎨 Graphismes** dans la page : objets (tuiles `$00…`, sprites `$80…`/`$C0…`), pixels, palette,
  miroirs, remplissage, `.gfx` ouvrir/télécharger/→ `/storage`, planches PNG, image → tuiles+carte ; éditeur de
  **tilemap** (`.map`, → `/storage`, → BASIC `alloc`/`poke`/`data`) (S3-1, S3-2). Vérifié en navigateur :
  `.gfx` + `.map` édités affichés par `gload`/`tiledraw`/`sprite`.
- Snippets `vmode1` (attributs MDA) et `gload` (S3-4). Doc : `docs/GRAPHICS.md` ; ADR-001 précisé (4 bpp).

## [0.3.0] - 2026-09-20
### Ajouté — sprint 2, compilateur NeoBASIC → 65C02 (fondations)
- `internal/asm` : assembleur 65C02 programmatique (étiquettes, rétro-correction, relaxation des branches,
  listing 64tass) — **719 octets identiques à 64tass** sur tous les mnémoniques et modes (S2-2).
- `internal/neo` : emballage `.neo` (oracle `mkneo.py`) ; `internal/neobasic.Lex`/`Encode` (éléments
  structurés, différentiel inchangé).
- `internal/compiler` : parseur sur le flux d'éléments (priorités de la table des tokens), générateur et
  runtime sur l'API `$FF00` — entiers 32 bits, chaînes, `print`, `if`, `while`, `repeat`, `do/exit/loop`,
  `for`, `proc`/`call`/`local`, `poke`/`doke`/`peek(`/`deek(`, `abs( sgn( min( max( rand( len( asc( chr$( str$(`
  (S2-1, S2-3). Périmètre et conventions : `docs/COMPILER.md` ; conception : `docs/adr/ADR-002`.
- Harnais différentiel (`make test-emu`) : **7 programmes du corpus produisent le même écran** interprété
  et compilé dans Phosphoneo ; le listing de chaque programme est ré-assemblé par 64tass (S2-5). Faits
  relevés sur l'interpréteur : `else` interdit après `if … then` sur une ligne, `exit` seulement dans
  `do … loop`, `proc` après `end`, `for` exécute le corps au moins une fois, constantes tronquées à 32 bits.
- `neoforgec` (CLI : `.neo`, `-bin`, `-list`), `POST /api/compile`, bouton **⚙ Compiler** (F6) dans la page
  (vérifié en navigateur) (S2-4).

## [0.2.0] - 2026-09-20
### Ajouté — onglets (S1-3)
- Onglets multi-programmes : un modèle Monaco par onglet, « ● » quand le contenu diffère de la version
  enregistrée, fermeture confirmée si non enregistré, Ouvrir/⬆ .bas/Nouveau ouvrent un onglet
  (ou réutilisent celui du même nom).
### Ajouté — fichiers de données (S1-7)
- 📎 Fichiers : envoie des fichiers (.gfx, niveaux, scores…) dans `/storage` de l'émulateur ; vérifié en
  navigateur avec la démo `crossdev` de Trinity (`examples/sprites.bsc` + `examples/graphics.gfx`,
  MIT © Paul Robson) : sprites, tiles et images animés.
### Ajouté — Trinity dans la page (S1-6, S1-8)
- Phosphoneo reconstruit contre `~/Neo6502Trinity` (voir son CHANGELOG du 2026-09-20 : gardes pour le
  code du fork, `TMRRead()` avance pendant l'attente active de `BOOTLoadChoice`, exports `web_type` et
  `web_reset`). Le firmware Trinity démarre sur NeoDOS : neoforge sert `NEOFORGE_NEOBASIC_BIN`
  (défaut `~/Neo6502Basic/bin/basic.bin`) sur `/emu-boot/neobasic.bin` et la page l'écrit dans
  `/storage/boot/` avec `auto.txt` avant le démarrage (`Module.preRun`).
- ⟳ Reset (`web_reset` : firmware + 65C02, `boot/auto.txt` rejoué) ; ■ Stop passe par `web_type("\\e")`.
- `vmode 1` (Hercules 720×350, 80 colonnes) vérifié dans la page ; `make e2e` prépare `storage/boot`.
### Corrigé — exécution dans le navigateur (S1-2)
- `phosphoneo.js` résolvait `phosphoneo.data`/`.wasm` relativement à la page (404) : `Module.locateFile`
  les renvoie sous `/emu/`.
- Le programme par défaut contenait un « — » non ASCII, refusé par le tokeniseur.
### Ajouté
- ■ Stop : envoie Échap (Break de NeoBASIC) à l'émulateur (S1-6, vérifié en navigateur :
  « Break Pressed at line 120 »).
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
