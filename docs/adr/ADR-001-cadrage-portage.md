# ADR-001 — Cadrage du portage de la Forge Oric sur Neo6502

Date : 2026-09-20 · État : **accepté** (décisions de bmarty, session de cadrage)

## Contexte

`~/forge` (oriced) est un IDE BASIC Oric : serveur Go + Monaco, émulateur Phosphoric intégré par flux
MJPEG, compilateur BASIC → 6502 (≈ 11 000 lignes hors tests) dont le runtime s'appuie sur la ROM Oric,
éditeurs HIRES/sprites. Objectif : la même expérience pour le **Neo6502** (NeoBASIC, firmware Trinity).

## Décisions

1. **Modes vidéo** = `vmode 0` (320×240, 256 couleurs, sprites/tilemaps) et `vmode 1` (Hercules 720×350,
   80 colonnes) de Trinity. Le « mode 2 » retiré de Trinity n'est pas réintroduit.
2. **Compilateur porté dès le début** (épopée E5) : nouveau backend runtime sur l'API firmware `$FF00`
   (au lieu des routines ROM Oric) ; frontend et assembleur d'oriced réutilisés.
3. **Sprites et tiles** : les éditeurs graphiques ciblent les formats Neo6502 (sprites 16×16/32×32 et
   tuiles 16×16 du mode 0, fichier `graphics.gfx` chargé par `gload`, tilemaps). Précision apportée au
   sprint 3 : ces objets sont en **4 bits par pixel, 16 couleurs de la palette du firmware** (le « 256
   couleurs » ne concerne que l'écran), cf. `gconvert.py`/`reference/graphics.md`.
4. **Émulateur = Phosphoneo WebAssembly dans la page** (cible `make wasm` existante) : pas de flux
   MJPEG ni d'API HTTP à ajouter à Phosphoneo, latence nulle, déploiement d'un seul binaire + un dossier.

## Conséquences

- Pas de fork d'oriced : nouveau module `github.com/bmarty/neoforge`, paquets repris un à un quand ils
  sont portés (chaque reprise = un récit testé), pour ne jamais embarquer de sémantique Oric morte.
- La tokenisation NeoBASIC est réécrite en Go **à l'identique** de `makebasic.py` (oracle différentiel)
  plutôt que d'appeler Python à l'exécution.
- Le compilateur est le chantier le plus long ; il démarre après le socle (sprint 1) pour disposer
  de l'exécution interprétée comme oracle de fidélité (comme la campagne E7 d'oriced).
