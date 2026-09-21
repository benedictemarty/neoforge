# Compilateur NeoBASIC → 65C02 (neoforgec)

Conception : `docs/adr/ADR-002-compilateur.md`. État : **sprint 9** — **48 des 50 exemples `.bsc` de Trinity compilent** (`tools/corpus.sh`) et
**36 donnent le même écran que l'interpréteur** (`tools/corpus_run.sh` ; les 12 autres affichent des
chronométrages ou des adresses `alloc(`). Restent : Sweet16 (`mixedassembler.bsc`, absent de NeoBASIC)
et `asteroids.bsc` (bibliothèque non incluse par le Makefile officiel). Performance : `tools/bench.sh`.

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
| `print` (`;` `,` taquets de 8, nombres via 4,34), `input` (saisie de 80 caractères avec écho, conversion 4,33, « ?? » et relecture si invalide, comme l'interpréteur), `cls`, `poke`/`doke`, `peek(`/`deek(` ; **fichiers** `open input/output`, `close`, `print #` / `input #` (enregistrements de l'interpréteur : nombre = `$FF` + type + 4 octets, chaîne = longueur + caractères), `eof(`, `save "f",adr,taille`, **`input line #` / `print line #`** (lignes LF, tabulation → espace, 252 caractères max ; fin de fichier = fin de ligne), **`cat [motif$]`** (3,1 / 3,32) | — |
| `if … then …` (une ligne, sans `else`), `if … / else / endif`, `while/wend`, `repeat/until`, `do/exit/loop`, `for … to/downto … next` | `on error` |
| `proc`/`endproc` (paramètres par valeur ou `ref` : recopie dans la variable de l'appelant au retour), `call`, `local` (entiers et chaînes), `data`/`read`/`restore`, `assert`, `defchr`, `load "f",adr`, `sys` (A, X, Y) ; **tableaux** `dim a(n[,m])` (1 ou 2 dimensions, bornes incluses, taille dynamique, chaînes comprises), **`goto`/`gosub`/`return`** vers des numéros de ligne constants | récursion, `case`/`when` (« Not Implemented » dans NeoBASIC) |
| `abs( sgn( int( min( max( rand( alloc( true false` ; opérateur `mem[i]` (mot 16 bits) ; **souris** `mouse to/show/hide/cursor`, `mouse(x,y[,w])`, `havemouse(` ; **GPIO/I2C** `pin n,input|output|analog|valeur`, `pin(`, `analog(`, `iwrite`, `iread(` ; **tortue** `turtle home|fast|hide|show`, `penup`, `pendown [c]`, `left`/`right`/`forward` (groupe 9, sprite $7F, temporisations de `turtle.asm`) | — |
| **UART/I2C/SPI** : `uconfig`, `usend`/`ssend`/`isend` (blocs : octet bas, `;` = mot, chaînes), `ureceive`/`utransmit`/`sreceive`/`stransmit`/`ireceive`/`itransmit`, `uhasdata(` ; `exists(` ; récursion (paramètres sauvegardés/restaurés) ; terminateur orphelin = « Structure Imbalance » à l'exécution | — |
| **assembleur en ligne** : mnémoniques 65C02 assemblés à l'exécution dans `P` (options `O`), étiquettes `.nom`, tous les modes (immédiat, page zéro/absolu choisi sur la valeur, `,x`/`,y`, indirects), branches relatives, listing hexadécimal (`O` bit 1) — code identique à celui de l'interpréteur | Sweet16 : absent de NeoBASIC actuel (`mixedassembler.bsc` obsolète, « Syntax Error » dans l'interpréteur lui-même) |
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
  piles, `$2E` compteur, `$30-$39` registres maths de l'API (entrelacés au pas 2), `$3C`/`$3D` types,
  `$3E` tas, `$40` pointeur data, `$42` pointeur des accès variables (mode compact).
- **Deux modes de génération.** Rapide (par défaut) : accès aux variables en ligne (30 octets par
  chargement). Si la fin du programme (`ENDPROG`, début du tas) dépasse `$E000`, le programme est
  recompilé en mode **compact** : variables par routines `RT_LDV`/`RT_STV`/`RT_LDT` (`ldx/ldy` + `jsr`,
  7 octets), constantes 0-255 par `RT_LDI8`/`RT_LTI8`, `+`/`-` et copies de registres par routines —
  environ deux fois plus petit (Atic Atac : 61 → 31 Ko), au prix de quelques cycles par accès.
  Au-delà de `$FE00` en compact : erreur « programme trop grand ». `neoforgec` et ⚙ Compiler
  annoncent la fin du programme et le mode. Les appels API passent toujours par `RT_API` (A = fonction,
  X = groupe) et `RT_MATH` (7 et 5 octets au lieu de 16 et 33).
- **Erreurs d'exécution** reproduites avec le numéro de ligne de la numérotation automatique (100, pas
  10, ou numéro explicite) : « File I/O Error at line N » (`load`, `gload`, `open`), « Division By Zero
  Error » (`/ \ %`), « Out Of Range Error » (indice de tableau hors de 0…borne), « Out Of Data »
  (`read` après le dernier `data`), « String Too Long » (concaténation dont le résultat dépasse 251
  caractères, règle de `mathstd.asm`), « Out Of Range Error » pour `chr$(` hors 0-255, `sqr(`/`log(`/
  `pow(`… en erreur API, `dim` (dimension 255, ou éléments × 5 ≥ 13056 — « cpy #51 » de `dim.asm`),
  « Out Of Memory » quand le tas (`alloc(`, `dim`) dépasse `$FE00` (l'interpréteur a moins de mémoire :
  `alloc(60000)` y échoue, pas ici) — message, CR, arrêt, comme `ErrorHandler` de l'interpréteur.
  Bug de l'interpréteur trouvé par ce différentiel et **corrigé dans Neo6502Basic** (commit `c1fe87c`,
  `concrete.asm` : une chaîne qui grandissait sur place débordait de 2 octets sur la chaîne concrétisée
  voisine — `b$ = "ab"` puis `a$ = a$ + "xyzw"` ×3 donnait `len(b$)` = 119) ; `strings2.bsc` couvre le motif.
- Expressions : TMP ← gauche, ACC ← droite, opération TMP ∘ ACC. Une feuille (constante, variable) est
  chargée directement ; sinon ACC ← gauche, `PUSH`, ACC ← droite, `POP` → TMP. `* \ %` passent par
  l'API maths (4,2 4,4 4,5) — mesuré plus rapide que des routines 32 bits natives (S8-1). Les
  comparaisons entières sont natives (`CMP32` : signe de la différence 32 bits **sans** correction de
  débordement, exactement comme `compare.asm` de l'interpréteur, où `2147483647 > -5` est faux) ; flottantes
  ou incertaines : 4,6. Une condition (`if`, `while`, `until`) se branche directement sur le résultat
  $FF/0/1 ; `for` incrémente l'indice sur place et compare par `CMP32`.
- Console : 2,6 (caractère), 2,12 (`cls`), 2,13 (position pour `,`), 4,34 (nombre → chaîne).

## Validation

- `make test` : parseur/générateur (100 % de couverture), listing de chaque programme du corpus assemblé par
  **64tass** avec les mêmes octets (`TestCorpusAssembles`).
- `make test-emu` (`NEOFORGE_EMU_TESTS=1`) : **différentiel** — chaque `internal/compiler/testdata/*.bsc`
  produit le même écran interprété (NeoBASIC à `$800` + `.bas`) et compilé (`.neo`) dans Phosphoneo.
