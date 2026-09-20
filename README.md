# neoforge

**Éditeur NeoBASIC moderne pour Neo6502** — portage de la Forge Oric ([oriced](../forge)) sur le
Neo6502 (Olimex, W65C02S + RP2040) sous firmware **Trinity**.

[![Licence: EUPL 1.2](https://img.shields.io/badge/licence-EUPL--1.2-blue.svg)](LICENSE)

`neoforge` est un serveur Go qui embarque l'éditeur **Monaco** (moteur de VS Code) avec un langage
**NeoBASIC** (coloration, complétion, aide au survol) et l'émulateur **[Phosphoneo](../Phosphoneo)**
compilé en **WebAssembly** : on écrit à gauche, ▶ Exécuter tokenise le source et le lance *dans la
page*, à droite, sur le vrai firmware Neo6502.

> État : **sprint 1 livré (v0.2.0)** — socle, tokeniseur/détokeniseur NeoBASIC, éditeur à onglets + aide,
> émulateur Trinity dans la page (Stop/Reset, `vmode 1`, fichiers de données), CLI `neobas`.
> Voir `docs/BACKLOG.md` (épopées, sprints) et `CHANGELOG.md`.

## Architecture en un coup d'œil

```
Navigateur : Monaco (édition NeoBASIC)      Phosphoneo WASM (65C02 + firmware Trinity, canvas)
        │ POST /api/build {source}                  ▲ Module.FS /storage/prog.bas + web_load_neo
        ▼                                           │
Serveur Go (neoforge) ── internal/neobasic ── prog.bas (tokenisé, = makebasic.py) ──┘
                        ── /emu/ ← ~/Phosphoneo/web (phosphoneo.js/.wasm/.data)
```

Voir `docs/ARCHITECTURE.md`.

## Prérequis

- Go ≥ 1.26 ; `node` pour les tests JS ; `python3` pour le test différentiel (optionnel).
- **Phosphoneo** construit en WebAssembly **contre le firmware Trinity** : `make -C ~/Phosphoneo wasm`
  (emsdk) → `~/Phosphoneo/web/`. Sans lui, l'éditeur fonctionne (tokenisation, `.bas` téléchargeable)
  mais l'écran reste vide.
- **NeoBASIC** (`~/Neo6502Basic/bin/basic.bin`, `make` dans ce dépôt) : Trinity démarre sur NeoDOS et
  lance NeoBASIC depuis `boot/neobasic.bin`, que neoforge place dans le stockage de l'émulateur.

## Démarrer

```bash
make run          # http://127.0.0.1:8098
```

- **▶ Exécuter** (F5) : tokenise le source ; la ligne en erreur est soulignée ; sinon le `.bas` est écrit
  dans le stockage de l'émulateur et lancé (`load "prog.bas"` + `run`).
- **■ Stop** : envoie Échap à l'émulateur (Break de NeoBASIC) ; **⟳ Reset** : redémarrage à froid
  (firmware + 65C02, retour à NeoBASIC via `boot/auto.txt`).
- **⬇ .bas** : télécharge le programme tokenisé, à copier sur la clé USB d'un Neo6502 réel ;
  **⬆ .bas** : importe un `.bas` (détokenisé dans l'éditeur).
- **Onglets** : plusieurs programmes ouverts ; « ● » signale un onglet non enregistré.
- **Ouvrir / Enregistrer** (Ctrl+S) : programmes `.bsc` de `NEOFORGE_PROJECTS_DIR`.
- **⌨ Clavier → Neo** : donne le clavier à l'émulateur (ou cliquer sur l'écran).
- **📎 Fichiers** : envoie des fichiers de données (`.gfx`, niveaux…) dans `/storage` de l'émulateur
  (essayer `examples/sprites.bsc` avec `examples/graphics.gfx`).
- **❔ Aide** (F1) : référence des commandes (documentation officielle neo6502-documents, MIT) filtrable,
  clic pour insérer ; la même aide apparaît au survol et dans la complétion.

## Configuration (variables d'environnement)

| Variable                  | Défaut              | Rôle                                            |
|---------------------------|---------------------|-------------------------------------------------|
| `NEOFORGE_ADDR`           | `127.0.0.1:8098`    | Adresse d'écoute HTTP                           |
| `NEOFORGE_PHOSPHONEO_WEB` | `~/Phosphoneo/web`  | Build WASM de Phosphoneo (servi sous `/emu/`)   |
| `NEOFORGE_PROJECTS_DIR`   | `~/NeoPrograms`     | Programmes `.bsc` (Ouvrir/Enregistrer)          |
| `NEOFORGE_NEOBASIC_BIN`   | `~/Neo6502Basic/bin/basic.bin` | NeoBASIC injecté dans `/storage/boot/` (Trinity démarre sur NeoDOS) |

## Outil en ligne de commande

```bash
go build -o neobas ./cmd/neobas
./neobas -o hello.bas examples/hello.bsc     # équivalent de makebasic.py (identique octet pour octet)
./neobas -library -o lib.bas lib.bsc         # bibliothèque (numéros de ligne à 0)
./neobas -list hello.bas                     # détokenise (= listbasic.py) ; -n sans numéros
```

## Développement

```bash
make test         # tests Go
make test-js      # logique de l'éditeur (node --test)
make cover-check  # porte : 100 % de couverture
make vet fmt
make e2e          # neobas → Phosphoneo natif → écran vérifié
```

La tokenisation est vérifiée en **différentiel** contre `makebasic.py` (75 programmes `.bsc` du firmware,
identiques octet pour octet) — test sauté si les scripts ou `python3` sont absents.

## Licence

**EUPL-1.2** — voir `LICENSE`. © bmarty.
