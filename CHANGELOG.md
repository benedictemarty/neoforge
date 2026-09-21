# Changelog

Toutes les modifications notables de **neoforge** sont consignées ici.
Format inspiré de [Keep a Changelog](https://keepachangelog.com/fr/1.1.0/),
versionnage [SemVer](https://semver.org/lang/fr/).

## [Non publié]
### Ajouté — compilateur : erreurs d'exécution et différentiel sur les exemples officiels (S9-1, S9-2)
- `tools/corpus_run.sh` : chaque exemple de Trinity est exécuté interprété et compilé, écrans comparés —
  36/48 identiques, 12 différences expliquées (chronométrages, adresses `alloc(`).
- Erreurs d'exécution avec numéro de ligne comme l'interpréteur : « File I/O Error at line N » (`load`,
  `gload`, `open` — les exemples sans `graphics.gfx` s'arrêtent désormais comme interprétés), « Division By
  Zero Error », « Out Of Range Error » (indices de tableau, bornes mémorisées à `dim`), « Out Of Data ».
  Différentiels `errdiv.bsc`, `errrange.bsc`, `errdata.bsc`, `files.bsc` (21 programmes).
### Modifié — compilateur : taille du code, mode compact automatique (S9-3)
- Appels API par `RT_API` (A = fonction, X = groupe) et `RT_MATH` ; copies de registres maths par
  routines en mode compact. Au-delà de `$E000`, recompilation en **mode compact** (variables par
  `RT_LDV`/`RT_STV`/`RT_LDT`, constantes `RT_LDI8`, `+`/`-` par routines) : Atic Atac 61 753 → 31 335
  octets — il écrasait la page zéro avec son tas (retour à NeoDOS). Erreur « programme trop grand »
  au-delà de `$FE00`. `neoforgec` et ⚙ Compiler affichent la fin du programme et le mode ; le
  différentiel tourne dans les deux modes. Bench : 205 → 214 (routines d'appel API, contrôles de division).

## [0.9.0] - 2026-09-21 — sprint 8 : performance, tortue, paquet de release
### Ajouté — `make dist` (S8-4)
- Paquet autonome `dist/neoforge-<version>-<os>-<arch>.tar.gz` : `neoforge`, `neobas`, `neoforgec`, `emu/`
  (Phosphoneo WASM), `boot/neobasic.bin`, docs et exemples. La configuration privilégie `emu/` et `boot/`
  placés à côté de l'exécutable (vérifié avec `HOME` inexistant : émulateur et NeoBASIC détectés).
### Ajouté — compilateur : tortue, `cat`, lignes de fichier (S8-3)
- `turtle home|fast|hide|show`, `penup`, `pendown [couleur]`, `left`/`right`/`forward` (groupe 9, sprite $7F,
  initialisation implicite et temporisations comme `turtle.asm`) ; `cat [motif$]` (3,1 / 3,32) ;
  `input line #canal, a$…` / `print line #canal, s$…` (LF, tabulation → espace, 252 caractères max).
  Nouveau différentiel `turtle.bsc` (texte et image identiques), `files.bsc` étendu ; 18 différentiels.
### Constat — Sweet16 (S8-2)
- Sweet16 n'existe pas dans NeoBASIC actuel (aucun token, aucune source) ; `mixedassembler.bsc` provoque
  « Syntax Error » dans l'interpréteur lui-même : exemple obsolète, hors périmètre (48/50 reste le maximum).
### Modifié — compilateur : performance du code généré (S8-1)
- Comparaisons entières natives (`RT_CMP32`, reproduit `compare.asm` : différence 32 bits sans correction de
  débordement), branchement direct des conditions `if`/`while`/`until` sur le résultat $FF/0/1, opérandes
  feuilles chargés sans pile, indice de `for` incrémenté/décrémenté sur place. `tools/bench.sh`
  (`tools/bench/bench.bsc`) : total 795 → 205 centièmes (interprété → compilé, ×3,9 ; boucle entière 439 → 87).
- Essayé puis retiré : `* \ %` en routines 32 bits natives — 5 fois plus lents que l'API 4,2/4,4/4,5 dans
  l'émulateur ; conservés en API. Différentiels `loops.bsc` (emprunts sur 16/24 bits, bornes négatives) et
  `compare.bsc` (flottants, chaînes, extrêmes) étendus.

## [0.8.0] - 2026-09-21 — sprint 7 : finitions du compilateur
### Ajouté — 📡 Carte : .neo compilé (S7-4)
- Le bouton 📡 Carte propose le `.neo` compilé ou le `.bas` ; transfert binaire vérifié dans l'émulateur
  (1 451 octets identiques). Déroulé de validation sur carte réelle dans `docs/CARD.md` (S7-5, matériel).
### Ajouté — compilateur : UART/I2C/SPI (S7-3)
- `uconfig`, `usend`/`ssend`/`isend` (tampon : octet bas, « ; » ajoute l'octet haut, chaînes), `ureceive`/
  `utransmit`/`sreceive`/`stransmit`/`ireceive`/`itransmit`, `uhasdata(`, `exists(` ; un terminateur orphelin
  (`next` sans `for`…) compile en arrêt « Structure Imbalance » à l'exécution, comme l'interpréteur.
  **48/50 exemples de Trinity compilent.**
### Modifié — compilateur : procédures récursives (S7-2)
- Les paramètres de procédure sont sauvegardés à l'appel et restaurés à `endproc` (fait relevé sur
  l'interpréteur : `local` implicite, variables globales pendant l'exécution) ; les paramètres `ref` ne
  sont pas restaurés. `recursion.bsc` (factorielle, Fibonacci, Hanoï) identique ; 16 différentiels.
- `library` est une commande d'éditeur (rien à compiler) ; `asteroids.bsc` appelle `Machine.Code()` d'une
  bibliothèque que le Makefile officiel n'inclut pas (exemple incomplet, hors périmètre).

## [0.7.0] - 2026-09-21 — sprint 6 : périphériques, carte réelle
### Ajouté — 📡 Carte (S6-3)
- `POST /api/xfer` (dépôt), `GET /api/xfer/{nom}?c=N` (tranches de 200 octets), `GET /api/xfer/{nom}/size`,
  `lanAddr` dans `/api/config` ; bouton **📡 Carte** : dépose le `.bas` et ouvre `recv.bsc`, récepteur
  NeoBASIC généré (`atconnect` optionnel, `atget$(` par tranches, `poke`, `save`). Le modem logiciel de
  Phosphoneo (`neomodem.js`) est attaché dans la page : le récepteur validé dans l'émulateur (fichier reçu
  identique). Doc : `docs/CARD.md`.
### Ajouté — compilateur : souris, GPIO, I2C (S6-1)
- `mouse to/show/hide/cursor` (11,1/11,2/11,5), `mouse(x,y[,w])` (11,3, par référence, 16 bits comme
  l'interpréteur), `havemouse(` (11,4), `pin n,input|output|analog|valeur` (10,4/10,2), `pin(` (10,3), `analog(`
  (10,7), `iwrite` (10,5), `iread(` (10,6) ; 44/50 exemples de Trinity compilent. Divergence : sur GPIO
  absent l'interpréteur signale « Out Of Range », le compilé continue.

## [0.6.0] - 2026-09-21 — sprint 5 : assembleur en ligne, fichiers, ref, débogueur
### Ajouté — débogueur (S5-4)
- Panneau 🐞 sous l'éditeur : ⏸ Pause / ▶ Continuer / ⏭ Pas, registres (PC + étiquette la plus proche, A, X,
  Y, S, indicateurs, cycles), **variables du programme compilé** (symboles renvoyés par `/api/compile`,
  nombres décodés selon leur type, chaînes), mémoire (64 octets hexa + ASCII). Exports Phosphoneo
  `web_pause`/`web_resume`/`web_step`/`web_regs`/`web_peek`/`web_poke` (commit Phosphoneo du 2026-09-21).
  Doc : `docs/DEBUGGER.md`.
### Ajouté — compilateur : ref, wait (S5-3)
- Paramètres `ref` (valeur recopiée dans la variable de l'appelant au retour de la procédure), `wait n`
  (boucle sur l'horloge 1,1 comme `wait.asm`) ; 40/50 exemples de Trinity compilent, Atic Atac compilé
  tourne dans Phosphoneo.
### Ajouté — compilateur : fichiers (S5-2)
- `open input|output canal,"nom"` (3,4), `close` (3,5), `print #`/`input #` (enregistrements typés octet par
  octet comme `inputprintfile.asm`, variables et éléments de tableau), `eof(` (3,22), `save "nom",adresse,taille`
  (3,3) ; 15 différentiels identiques ; 39/50 exemples de Trinity compilent.
### Ajouté — compilateur : assembleur en ligne (S5-1)
- Mnémoniques 65C02 assemblés à l'exécution dans la variable `P` (`O` : bit 0 dernière passe, bit 1
  listing hexadécimal), étiquettes `.nom` (fixées si la variable vaut 0), choix page zéro/absolu sur la
  valeur de l'opérande, `,x`/`,y`, modes indirects, branches relatives à `P` — mécanisme de
  `assembler/*.asm` reproduit ; opérateur `mem[i]` (lecture/écriture 16 bits). Code assemblé **identique
  octet pour octet** à l'interpréteur (`inline.bsc` : taille et somme de contrôle) ; 14 différentiels.
- Fait relevé : les modes indirects `(zp),y`, `(zp,x)`, `(zp)` exigent un opérande < 256 (erreur sinon).

## [0.5.0] - 2026-09-21 — sprint 4 : compilateur, suite
### Ajouté — corpus (S4-5)
- `data`/`read`/`restore` (pool global typé), `assert`, `defchr`, `load "f",adresse`, `sys` (A, X, Y),
  `local` sur les chaînes, `event(` sur un élément de tableau, déclarations `dim` visibles de tout le
  programme, opérateurs bit à bit sur des nombres non prouvés entiers.
- `tools/corpus.sh` : **34 des 50 exemples `.bsc` de Trinity compilent** (restent l'assembleur en ligne, les
  fichiers, `mouse`/`pin`/`i2c`/`uconfig`, `ref`) ; Breakout compilé tourne dans Phosphoneo.
- Différentiel : 13 programmes identiques texte et image (`data.bsc` ajouté, `level.map` fourni au stockage).
### Ajouté — compilateur : tableaux et goto (S4-3)
- `dim a(n[,m])` (bornes incluses, tailles dynamiques, tableaux de chaînes), éléments en lecture/écriture/`input`,
  `goto`/`gosub`/`return` vers des lignes numérotées (marqueurs de ligne conservés par le parseur, cibles
  vérifiées à la compilation). Différentiel : 12 programmes identiques (texte et image).
- Fait relevé : `case`/`when` répond « Not Implemented » dans NeoBASIC — non implémenté.
### Ajouté — compilateur : matériel (S4-4)
- Commandes graphiques chaînées (`move line rect ellipse plot text image tiledraw` + `from to by ink solid frame
  dim`), `sprite` (héritage du bloc entre sprites d'une même commande, comme l'interpréteur), `gload`, `tilemap`,
  `sound`/`noise`/`sfx`, `vmode`, `ink` (avec la particularité de `ink.asm`), `cursor`, `palette`, `alloc(`, et les
  fonctions `event( joypad( time( vblanks( key( vmode( notes( point( spoint( hit( spritex( spritey(` — mêmes
  appels API que l'interpréteur.
- Harnais différentiel : comparaison des **captures d'écran** en plus du texte (curseur de l'invite exclu),
  fichiers de données (`graphics.gfx`) fournis au stockage ; 11 programmes identiques texte et image.
### Ajouté — compilateur : flottants (S4-2)
- Nombres à type dynamique (octet de type comme les registres de l'API), variables numériques sur 5 octets,
  inférence par point fixe des variables entières (chemin 65C02 natif conservé), opérations et fonctions
  flottantes par le groupe 4 (`/`, `sin cos tan atan atan2 log exp sqr pow rnd int abs sgn`), constantes
  décimales converties par la formule du firmware. Différentiel : 10 programmes identiques (dont `floats.bsc`).
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
