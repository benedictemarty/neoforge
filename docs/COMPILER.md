# Compilateur NeoBASIC → 65C02 (neoforgec)

Conception : `docs/adr/ADR-002-compilateur.md`. État : **sprint 6 en cours** — S6-1 E/S matérielles livré. **44 des 50 exemples `.bsc` de Trinity compilent**
(`tools/corpus.sh`) ; Breakout et Atic Atac (jeux officiels) tournent compilés. Restent : `uconfig`/`usend`/
`itransmit`… (blocs UART/I2C/SPI), `library`, Sweet16 (et un exemple bogué).

## Utilisation

```bash
neoforgec prog.bsc              # → prog.neo (exécutable : load "prog.neo" sous NeoBASIC, menu boot/, Phosphoneo)
neoforgec -bin -o prog.bin p.bsc  # binaire brut chargé en $800
neoforgec -list prog.bsc        # listing 64tass du code généré
```

Dans l'IDE : **⚙ Compiler** (F6) compile et lance le `.neo` dans l'émulateur ; la ligne fautive est soulignée.

## Périmètre (sprint 2)

| Pris en charge | Hors périmètre (sprints suivants) |
|---|---|
| entiers 32 bits signés (`+ - * \ % & \| ^ << >>`, comparaisons → -1/0, `not` logique, `-` unaire) ; **flottants** simple précision (constantes décimales, `/`, `sin( cos( tan( atan( atan2( log( exp( sqr( pow( rnd( int( abs( sgn(`, mixte entier/flottant) | `& \| ^ << >>` sur un flottant |
| chaînes : constantes, variables (`x$`, 255 caractères), `+`, comparaisons (`= <> < > <= >=`, lexicographiques → -1/0), `len( asc( chr$( str$( left$( right$( mid$( instr( val( isval( upper$( lower$( spc( inkey$(` | `tab(`, `key(`, `event(` |
| `print` (`;` `,` taquets de 8, nombres via 4,34), `input` (saisie de 80 caractères avec écho, conversion 4,33, « ?? » et relecture si invalide, comme l'interpréteur), `cls`, `poke`/`doke`, `peek(`/`deek(` ; **fichiers** `open input/output`, `close`, `print #` / `input #` (enregistrements de l'interpréteur : nombre = `$FF` + type + 4 octets, chaîne = longueur + caractères), `eof(`, `save "f",adr,taille` | `input line #`, `cat`, `exists(` |
| `if … then …` (une ligne, sans `else`), `if … / else / endif`, `while/wend`, `repeat/until`, `do/exit/loop`, `for … to/downto … next` | `on error` |
| `proc`/`endproc` (paramètres par valeur ou `ref` : recopie dans la variable de l'appelant au retour), `call`, `local` (entiers et chaînes), `data`/`read`/`restore`, `assert`, `defchr`, `load "f",adr`, `sys` (A, X, Y) ; **tableaux** `dim a(n[,m])` (1 ou 2 dimensions, bornes incluses, taille dynamique, chaînes comprises), **`goto`/`gosub`/`return`** vers des numéros de ligne constants | récursion, `case`/`when` (« Not Implemented » dans NeoBASIC) |
| `abs( sgn( int( min( max( rand( alloc( true false` ; opérateur `mem[i]` (mot 16 bits) ; **souris** `mouse to/show/hide/cursor`, `mouse(x,y[,w])`, `havemouse(` ; **GPIO/I2C** `pin n,input|output|analog|valeur`, `pin(`, `analog(`, `iwrite`, `iread(` | blocs `usend`/`isend`/`ssend`/`uconfig`, turtle |
| **assembleur en ligne** : mnémoniques 65C02 assemblés à l'exécution dans `P` (options `O`), étiquettes `.nom`, tous les modes (immédiat, page zéro/absolu choisi sur la valeur, `,x`/`,y`, indirects), branches relatives, listing hexadécimal (`O` bit 1) — code identique à celui de l'interpréteur | Sweet16 |
| **graphisme chaîné** `move line rect ellipse plot text image tiledraw` (`from to by x,y ink solid frame dim`), **`sprite`** (`image to by flip anchor hide`, `sprite clear`), `gload`, `tilemap`, `sound`/`noise`/`sfx`, `vmode`, `wait`, `ink`, `cursor`, `palette` ; fonctions `event( joypad( time( vblanks( key( vmode( notes( point( spoint( hit( spritex( spritey(` | `joypad(` à 3 arguments, `mouse(`, `frame`, fichiers |

Les programmes suivent les conventions de l'interpréteur (vérifiées sur Phosphoneo, non inventées) :
`proc` se définissent **après `end`** ; `exit` n'est valide **que dans `do … loop`** ; `for` exécute son corps
**au moins une fois** ; les constantes sont tronquées à 32 bits (`2147483648` → `-2147483648`) ; `\` tronque
vers zéro, `%` ignore les signes, les décalages sont logiques et `x << 33` = 0 ; `instr` et `mid$` sont en
base 1 (`instr(a$,"")` = 1, `mid$(a$,20)` = ""), `right$(a$,50)` = la chaîne entière.

Divergences assumées (l'interpréteur signale une erreur, le compilé continue) : `mid$(a$,0,…)` est traité
comme `mid$(a$,1,…)`, un argument négatif de `left$`/`right$`/`mid$`/`spc` vaut 0, `val(` d'un texte non
numérique vaut 0 ; les indices de tableau ne sont pas contrôlés (l'interpréteur signale « Out Of Range »).

Tableaux : alloués sur le tas à l'exécution de `dim` (éléments numériques de 5 octets, chaînes de 256 octets —
`dim s$(100)` occupe 25 Ko), adressés par `i × colonnes + j` (multiplication 16 bits), mis à zéro.

## Matériel : mêmes appels API que l'interpréteur

Chaque commande matérielle reproduit la séquence d'appels de `commands/hardware/*.asm` : état graphique
(position, mode, encre, solide, taille, flip, texte, image) tenu dans `GSTATE` et envoyé par 5,1 ; `to`/`by`
copient l'ancienne position en Param4-7, la nouvelle en Param0-3 puis appellent 5,mode ; bloc sprite de 8
octets (`$80` = inchangé) envoyé par 6,2 — un `sprite n` suivant dans la même commande hérite des champs du
précédent ; `event(` opère par référence sur la variable ; `joypad(dx,dy)` écrit -1/0/1 dans `dx`/`dy`.
Particularité reproduite : `ink c,p` écrit l'octet du token virgule (`$CA`) avant le code papier, comme
`ink.asm`. Le différentiel compare aussi les **captures d'écran** (PPM) des deux exécutions, curseur exclu.

## Nombres : entiers natifs, flottants par l'API

Chaque nombre porte un **type dynamique** (octet `$3C` pour ACC, `$3D` pour TMP, premier octet des variables :
0 entier, `$40` flottant — la convention des registres de l'API). Les opérations dont les deux opérandes sont
**prouvés entiers** (constante entière, variable jamais affectée d'une expression non entière ni saisie par
`input`, fonctions entières) sont compilées en 65C02 natif ; sinon `+ - * / \ %`, `-`, `abs sgn int` et les
fonctions passent par le groupe 4 de l'API, qui applique exactement les règles de l'interpréteur (résultat
flottant si un opérande l'est, `int(` = plancher, `sgn(` entier). L'inférence est un point fixe sur tout le
programme (affectations, `input`, arguments de `call`). `print`/`str$(` utilisent 4,34 : un flottant s'affiche
avec 6 décimales comme dans l'interpréteur. Les constantes décimales sont converties par la formule du firmware
(`float32(entier) + float32(chiffres)/float32(10^n)`).

## Code généré

- Chargé en **`$800`** ; prologue (piles), corps, `bra *` final, procédures (`PROC_x`), routines runtime
  utilisées (`RT_*`, fermeture des dépendances), constantes chaînes, variables (`VAR_x` : 4 octets ;
  `VARBUF_x$` : 256 octets), tampons de chaînes temporaires (un par nœud), `NBUF`, piles `STK`/`LSTK`.
- Page zéro : `$20` ACC (32 bits), `$24` TMP, `$28`/`$2A` pointeurs de chaînes, `$2C`/`$2D` indices de
  piles, `$2E` compteur, `$30-$39` registres maths de l'API (entrelacés au pas 2).
- Expressions : ACC ← gauche, `PUSH`, ACC ← droite, `POP` → TMP, opération TMP ∘ ACC. `* \ %` et les
  comparaisons passent par l'API maths (4,2 4,4 4,5 4,6) — mêmes résultats que l'interpréteur par construction.
- Console : 2,6 (caractère), 2,12 (`cls`), 2,13 (position pour `,`), 4,34 (nombre → chaîne).

## Validation

- `make test` : parseur/générateur (100 % de couverture), listing de chaque programme du corpus assemblé par
  **64tass** avec les mêmes octets (`TestCorpusAssembles`).
- `make test-emu` (`NEOFORGE_EMU_TESTS=1`) : **différentiel** — chaque `internal/compiler/testdata/*.bsc`
  produit le même écran interprété (NeoBASIC à `$800` + `.bas`) et compilé (`.neo`) dans Phosphoneo.
