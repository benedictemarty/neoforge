# Graphismes — sprites, tuiles, tilemaps (🎨)

État : **sprint 3 livré (v0.4.0)**. Références (rien d'inventé) : `gconvert.py`/`makeimg.py` du firmware,
`neo6502-documents/reference/graphics.md`, `documents/release/crossdev/`.

## Formats

- **`graphics.gfx`** (`gload "graphics.gfx"`) : en-tête de 256 octets `[1][n tuiles 16][n sprites 16][n sprites 32]`,
  puis les objets en **4 bits par pixel** (quartet haut = pixel gauche), couleurs 1-15 de la palette du firmware
  (PICO-8), **0 = transparent** pour les sprites. Numéros d'image : tuiles `$00-$7F` (128 max), sprites 16×16
  `$80-$BF` (64), sprites 32×32 `$C0-$FF` (64).
- **Planches PNG** de `makeimg` (`tile_16.png`, `sprite_16.png`, `sprite_32.png`) : bande palette de 16 px, cases
  de (taille+8) px avec gouttières de 8 px, 16 par rangée (8 pour 32×32), magenta (255,0,255) = transparent ; les
  cases vides (tuile toute à la couleur 8, sprite tout à 0) sont ignorées.
- **Tilemap** en mémoire : `[1][largeur][hauteur][numéros de tuiles…]` ; `$F0` = transparent, `$F1-$FF` = plein.
  Fichier `.map` chargeable par `load "level.map",adr` (`adr = alloc(l*h+3)`), puis `tilemap adr,x,y` et `tiledraw`.

## Éditeur (bouton 🎨 Graphismes)

- Liste des objets (vignettes, numéro d'image), `+ tuile`, `+ sprite 16`, `+ sprite 32`, 🗑.
- Éditeur de pixels : crayon, remplissage, miroirs ↔ ↕, effacer, palette (0 = transparent).
- Fichiers : 📂/⬇ `.gfx`, `→ /storage` (puis `gload` dans le programme), 📥/📤 planches PNG au format `makeimg`
  (interopérables avec les scripts officiels), 🖼 **image → tuiles + carte** (PNG/JPEG/GIF : couleurs ramenées à
  la palette, tuiles 16×16 dédupliquées jusqu'à 128, tilemap de la taille de l'image).
- Tilemap : taille, peinture (clic), prélèvement (clic droit), tuile sélectionnée / transparent / plein,
  📂/⬇ `.map`, `→ /storage`, `→ BASIC` (insère `alloc`/`poke`/`data` dans l'éditeur).

## Validation

`internal/gfx` : aller-retour du `graphics.gfx` livré ; **import des trois planches crossdev identique à la
sortie de `makeimg`** (exécuté en test) ; planche exportée réimportée à l'identique ; couverture 100 %.
Navigateur : `.gfx` et `.map` de l'éditeur envoyés dans `/storage`, affichés par `gload`/`tiledraw`/`sprite`.
