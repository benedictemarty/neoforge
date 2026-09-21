# Backlog produit — neoforge

Gestion agile. Priorité : **P1** (haute) → **P3** (basse). État : `TODO`, `EN COURS`, `FAIT`.

**État global (2026-09-21)** : sprints 0 à 7 **livrés** (v0.8.0), sprint 8 **en cours** (S8-1 à S8-3 faits) ; 48/50 exemples officiels compilent ; reste la validation sur carte réelle (S7-5, matériel requis) et les idées non planifiées. Cadrage : `docs/adr/ADR-001`.

**Dépendance Phosphoneo** : le WASM doit être construit contre `~/Neo6502Trinity` (`make -C ~/Phosphoneo wasm`, commits du 2026-09-20 : gardes fork, `TMRRead` en attente active, exports `web_type`/`web_reset`). Sous Trinity le firmware démarre sur NeoDOS ; neoforge injecte `boot/neobasic.bin` (`NEOFORGE_NEOBASIC_BIN`) dans le stockage de l'émulateur.

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

## Sprint 1 — Édition confortable & exécution fiable (v0.2.0) — FAIT

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S1-1 | E4    | Dépôt distant GitHub `benedictemarty/neoforge` + CI (gofmt, vet, cover-check, JS)   | P1   | FAIT |
| S1-2 | E2    | Vérifier le chargement à chaud WASM dans un vrai navigateur (`make e2e-browser`)    | P1   | FAIT |
| S1-3 | E1    | Onglets multi-programmes (un modèle Monaco par onglet), « ● » non enregistré, fermeture confirmée | P2   | FAIT |
| S1-4 | E1    | Détokeniseur `.bas` → source (`listbasic.py`) : ⬆ .bas, `neobas -list`, `/api/detok` | P2   | FAIT |
| S1-5 | E1    | Aide des commandes (panneau ❔/F1 filtrable, survol, complétion) depuis neo6502-documents | P2   | FAIT |
| S1-6 | E2    | ■ Stop (`web_type("\\e")` = Break) et ⟳ Reset (`web_reset` : firmware + 65C02, `boot/auto.txt` rejoué) — Phosphoneo reconstruit contre Trinity | P2   | FAIT |
| S1-7 | E3    | 📎 Fichiers : données du programme (.gfx, niveaux…) envoyées dans `/storage` (validé : démo sprites crossdev) | P2   | FAIT |
| S1-8 | E6    | Aperçu `vmode 1` (80 col.) : rendu Hercules vérifié dans la page (WASM Trinity)      | P2   | FAIT |

## Sprint 2 — Compilateur, fondations (v0.3.0) — FAIT

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S2-1 | E5    | Parseur NeoBASIC sur le flux d'éléments (`neobasic.Lex`), priorités de la table    | P1   | FAIT |
| S2-2 | E5    | Assembleur 65C02 (`internal/asm`) validé contre 64tass (719 octets, 66 mnémoniques) | P1   | FAIT |
| S2-3 | E5    | Runtime sur l'API `$FF00` : print, entiers 32 bits, chaînes, boucles, procs (`input` reporté) | P1   | FAIT |
| S2-4 | E5    | Emballage `.neo` (oracle `mkneo.py`), `neoforgec`, ⚙ Compiler dans la page        | P1   | FAIT |
| S2-5 | E9    | Harnais différentiel interprété/compilé (`make test-emu`, 7 programmes identiques) | P1   | FAIT |

## Sprint 3 — Sprites, tiles, modes (v0.4.0) — FAIT

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S3-1 | E7    | Éditeur de sprites 16×16/32×32 et tuiles (4 bpp, palette firmware), `.gfx`, planches PNG makeimg | P1   | FAIT |
| S3-2 | E7    | Éditeur de tilemap (`.map` chargeable par `load`, → BASIC alloc/poke/data)         | P1   | FAIT |
| S3-3 | E7    | Import d'image → tuiles dédupliquées + tilemap (validé en navigateur)              | P2   | FAIT |
| S3-4 | E6    | Snippets `vmode1` (attributs MDA) et `gload` (sprite + tilemap)                    | P3   | FAIT |

## Sprint 4 — Compilateur, suite (v0.5.0) — FAIT

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S4-1 | E5    | `input`, `inkey$(`, comparaisons de chaînes, `left$( right$( mid$( instr( val( isval( upper$( lower$( spc(` — 9/9 différentiels (frappe automatique `.keys`) | P1   | FAIT |
| S4-2 | E5    | Flottants : nombres à type dynamique, inférence des entiers (chemin natif), groupe 4 — 10/10 différentiels | P1   | FAIT |
| S4-3 | E5    | Tableaux `dim` 1D/2D (dynamiques, chaînes), `goto`/`gosub`/`return` ; `case/when` non implémenté dans NeoBASIC, `ref` reporté | P2   | FAIT |
| S4-4 | E5    | Graphisme chaîné, sprites, `gload`, `tilemap`, son, `vmode`, `ink`, `cursor`, `palette`, `event( joypad( hit(`… — différentiel texte **et image** 11/11 | P1   | FAIT |
| S4-5 | E9    | `tools/corpus.sh` : 34/50 exemples de Trinity compilent (+ `data`/`read`, `assert`, `defchr`, `load`, `sys`, `local` chaîne) ; 13 programmes différentiels | P2   | FAIT |

## Sprint 5 — Compilateur : assembleur, fichiers, débogueur (v0.6.0) — FAIT (S5-5 reporté)

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S5-1 | E5    | Assembleur en ligne (mnémoniques à l'exécution dans `P`/`O`, `.nom`, tous modes, `mem[i]`) — code identique à l'interpréteur ; 37/50 exemples | P2   | FAIT |
| S5-2 | E5    | Fichiers (`open`, `close`, `print #`, `input #`, `eof(`, `save`) — FAIT (39/50) ; `mouse`, `pin`, `i2c`, `uconfig` reportés | P3   | FAIT |
| S5-3 | E5    | Paramètres `ref` (copie entrée/sortie) + `wait` — FAIT ; récursion reportée (Atic Atac compile, 40/50) | P3   | FAIT |
| S5-4 | E8    | Débogueur 🐞 : pause/pas/reprise (exports Phosphoneo), registres, variables du programme compilé (symboles), mémoire | P2   | FAIT |
| S5-5 | E3    | Carte réelle : envoi du `.bas`/`.neo` par le modem Pico W                          | P3   | TODO (sprint 6) |

## Sprint 6 — Périphériques, carte réelle (v0.7.0) — FAIT (S6-2 reporté)

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S6-1 | E5    | `mouse` (commande et fonction), `havemouse(`, `pin`/`pin(`/`analog(`, `iwrite`/`iread(` — 44/50 exemples | P2   | FAIT |
| S6-2 | E5    | `library`, récursion, blocs `usend`/`isend`/`ssend`/`uconfig`, transfert du `.neo` compilé | P3   | TODO (sprint 7) |
| S6-3 | E3    | 📡 Carte : dépôt sur neoforge + récepteur NeoBASIC (`atget$(` par tranches, `save`) — validé dans l'émulateur via le modem logiciel ; carte réelle à valider | P3   | FAIT |

## Sprint 7 — Finitions du compilateur (v0.8.0) — FAIT (S7-5 à réaliser sur matériel)

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S7-1 | E5    | `library` : commande d'éditeur, rien à compiler ; `asteroids.bsc` incomplet côté firmware | P3   | FAIT (constat) |
| S7-2 | E5    | Récursion : paramètres sauvegardés/restaurés comme l'interpréteur                  | P3   | FAIT |
| S7-3 | E5    | Blocs UART/I2C/SPI, `exists(`, terminateurs orphelins — 48/50 exemples              | P3   | FAIT |
| S7-4 | E3    | 📡 Carte : choix `.neo` compilé / `.bas` — validé dans l'émulateur (1 451 octets identiques) | P3   | FAIT |
| S7-5 | E3    | Validation sur carte réelle (Neo6502 + modem Pico W) — déroulé dans `docs/CARD.md`, nécessite le matériel | P3   | TODO (bmarty) |

## Sprint 8 — Performance, compléments, livraison — EN COURS

Objectif : ce qui est faisable sans matériel. Mesure `tools/bench.sh` (Phosphoneo natif, `time()` en 1/100 s).

| ID   | Épop. | Récit utilisateur                                                                  | Prio | État |
|------|-------|------------------------------------------------------------------------------------|------|------|
| S8-1 | E5    | Code généré plus rapide : comparaisons entières natives (`CMP32`, reproduit la comparaison sans correction de débordement de `compare.asm`), branchement direct des conditions, opérandes feuilles sans pile, `for` incrémenté sur place — bench total 795 → 205 (×3,9 ; boucle entière ×5). Constat : `* \ %` natifs mesurés **plus lents** que l'API 4,2/4,4/4,5 dans l'émulateur (mul/div 48 → 255) → conservés en API ; latence de l'API sur carte réelle inconnue (non mesurée) | P2   | FAIT |
| S8-2 | E5    | Sweet16 : **absent de NeoBASIC actuel** (aucun token, aucune source ; `mixedassembler.bsc` donne « Syntax Error at line 200 » dans l'interpréteur lui-même) — exemple obsolète, rien à compiler | P3   | FAIT (constat) |
| S8-3 | E5    | `input line #`/`print line #`, tortue (groupe 9), `cat` — différentiels `turtle.bsc` (texte + image) et `files.bsc` étendus | P3   | FAIT |
| S8-4 | E1    | Paquets de release (`make dist` : binaire + web + émulateur)                        | P3   | TODO |

## Idées non planifiées

- Débogueur (E8) : Phosphoneo expose `--gdb`, `--tui`, `--break-api` en natif ; en WASM il faudra
  exporter des fonctions (`web_pause`, `web_regs`, `web_peek`) — à chiffrer avec Phosphoneo.
- Carte réelle : envoi du `.bas` par le modem Pico W (Neo6502Basic) au lieu de la clé USB.
