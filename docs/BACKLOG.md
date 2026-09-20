# Backlog produit — neoforge

Gestion agile. Priorité : **P1** (haute) → **P3** (basse). État : `TODO`, `EN COURS`, `FAIT`.

**État global (2026-09-20)** : sprint 0 **livré** (v0.1.0) ; sprint 1 ouvert. Cadrage : `docs/adr/ADR-001`.

## Épopées

- **E1 — Édition NeoBASIC moderne** (Monaco, coloration, complétion, diagnostics, aide)
- **E2 — Couplage émulateur** (Phosphoneo WASM dans la page, exécution, clavier, captures)
- **E3 — Gestion de fichiers & projets** (.bsc, .bas, .neo, clé USB, projets multi-fichiers)
- **E4 — Qualité & industrialisation** (tests, couverture 100 %, CI, packaging, releases)
- **E5 — Compilateur NeoBASIC → 65C02 natif** (frontend/asm repris d'oriced, runtime sur l'API `$FF00`)
- **E6 — Modes vidéo** (`vmode 0` 320×240/256 c. et `vmode 1` Hercules 80 col. : aperçus, éditeur adapté)
- **E7 — Sprites & tiles** (éditeurs sprites 16×16 / tilemaps, palette, export mémoire graphique `.gfx`/`.neo`)
- **E8 — Débogueur** (Phosphoneo : registres, mémoire, pas-à-pas, points d'arrêt BASIC)
- **E9 — Compatibilité & fidélité** (test différentiel interprété/compilé, corpus `.bsc` de Trinity)

## Sprint 0 — Socle (v0.1.0) — FAIT

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S0-1 | E4    | Structure projet Go + Makefile + git initialisé + licence EUPL                     | P1   | FAIT |
| S0-2 | E1    | Tokeniseur NeoBASIC en Go, identique à `makebasic.py` (différentiel sur 75 `.bsc`) | P1   | FAIT |
| S0-3 | E3    | CLI `neobas` (.bsc → .bas, `-library`)                                             | P2   | FAIT |
| S0-4 | E1    | Éditeur Monaco avec langage NeoBASIC (coloration, complétion, survol, snippets)    | P1   | FAIT |
| S0-5 | E2    | Phosphoneo WASM dans la page ; ▶ Exécuter = tokeniser + `load`/`run`               | P1   | FAIT |
| S0-6 | E3    | Ouvrir / Enregistrer des `.bsc` ; télécharger le `.bas`                            | P2   | FAIT |
| S0-7 | E4    | Tests Go 100 %, tests JS, `make e2e` (Phosphoneo natif), documentation             | P1   | FAIT |

## Sprint 1 — Édition confortable & exécution fiable (v0.2.0) — EN COURS

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S1-1 | E4    | Dépôt distant GitHub `benedictemarty/neoforge` + CI (gofmt, vet, cover-check, JS)   | P1   | FAIT |
| S1-2 | E2    | Vérifier le chargement à chaud WASM dans un vrai navigateur (capture, clavier)      | P1   | TODO |
| S1-3 | E1    | Onglets multi-programmes, indicateur « ● » non enregistré                          | P2   | TODO |
| S1-4 | E1    | Détokeniseur `.bas` → source (`listbasic.py`) : ouvrir un `.bas` de la clé         | P2   | TODO |
| S1-5 | E1    | Aide des commandes (panneau filtrable, clic pour insérer) depuis `basic.txt`        | P2   | TODO |
| S1-6 | E2    | ■ Stop / ⟳ Reset de l'émulateur, son (autoplay au geste ▶)                         | P2   | TODO |
| S1-7 | E3    | Fichiers de données du programme (.gfx, niveaux…) envoyés dans `/storage`          | P2   | TODO |
| S1-8 | E6    | Aperçu `vmode 1` (80 col.) : rendu canvas 720×350 correct dans la page             | P2   | TODO |

## Sprint 2 — Compilateur, fondations (v0.3.0) — TODO

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S2-1 | E5    | Reprise du frontend (lexeur/parseur) d'oriced adapté à la grammaire NeoBASIC       | P1   | TODO |
| S2-2 | E5    | Reprise de l'assembleur 6502 (+ opcodes 65C02 : BRA, PHX/PHY, STZ, TRB/TSB…)       | P1   | TODO |
| S2-3 | E5    | Runtime minimal sur l'API `$FF00` : PRINT, INPUT, entiers, chaînes, FOR/WHILE/IF   | P1   | TODO |
| S2-4 | E5    | Emballage `.neo` (`mkneo.py`) et exécution du binaire compilé dans Phosphoneo      | P1   | TODO |
| S2-5 | E9    | Harnais différentiel interprété/compilé sur `--screenshot-text`                    | P1   | TODO |

## Sprint 3 — Sprites, tiles, modes (v0.4.0) — TODO

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S3-1 | E7    | Éditeur de sprites 16×16 (palette 256 c.), export mémoire graphique                | P1   | TODO |
| S3-2 | E7    | Éditeur de tilemap (tiles 16×16, carte w×h), `tilemap`/`tiledraw` générés          | P1   | TODO |
| S3-3 | E7    | Import d'image → palette + tiles (comme l'import HIRES d'oriced)                   | P2   | TODO |
| S3-4 | E6    | Snippets/complétion spécifiques `vmode 1` (attributs MDA, 80×25)                   | P3   | TODO |

## Idées non planifiées

- Débogueur (E8) : Phosphoneo expose `--gdb`, `--tui`, `--break-api` en natif ; en WASM il faudra
  exporter des fonctions (`web_pause`, `web_regs`, `web_peek`) — à chiffrer avec Phosphoneo.
- Carte réelle : envoi du `.bas` par le modem Pico W (Neo6502Basic) au lieu de la clé USB.
