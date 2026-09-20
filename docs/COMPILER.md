# Compilateur NeoBASIC → 65C02 (neoforgec)

Conception : `docs/adr/ADR-002-compilateur.md`. État : **sprint 4 en cours** (S4-1 livré : entrées et chaînes).

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
| entiers 32 bits signés (`+ - * \ % & \| ^ << >>`, comparaisons → -1/0, `not` logique, `-` unaire) | flottants (`/`, constantes décimales, `sin(`…) |
| chaînes : constantes, variables (`x$`, 255 caractères), `+`, comparaisons (`= <> < > <= >=`, lexicographiques → -1/0), `len( asc( chr$( str$( left$( right$( mid$( instr( val( isval( upper$( lower$( spc( inkey$(` | `tab(`, `key(`, `event(` |
| `print` (`;` `,` taquets de 8, nombres via 4,34), `input` (saisie de 80 caractères avec écho, conversion 4,33, « ?? » et relecture si invalide, comme l'interpréteur), `cls`, `poke`/`doke`, `peek(`/`deek(` | fichiers (`#`), son |
| `if … then …` (une ligne, sans `else`), `if … / else / endif`, `while/wend`, `repeat/until`, `do/exit/loop`, `for … to/downto … next` | `goto`/`gosub`, `on error`, `case/when` |
| `proc`/`endproc` (paramètres par valeur), `call`, `local` (entiers) | `ref`, tableaux (`dim`), récursion |
| `abs( sgn( int( min( max( rand( true false` | graphisme, sprites, tilemaps, `event(`, assembleur `[ ]` |

Les programmes suivent les conventions de l'interpréteur (vérifiées sur Phosphoneo, non inventées) :
`proc` se définissent **après `end`** ; `exit` n'est valide **que dans `do … loop`** ; `for` exécute son corps
**au moins une fois** ; les constantes sont tronquées à 32 bits (`2147483648` → `-2147483648`) ; `\` tronque
vers zéro, `%` ignore les signes, les décalages sont logiques et `x << 33` = 0 ; `instr` et `mid$` sont en
base 1 (`instr(a$,"")` = 1, `mid$(a$,20)` = ""), `right$(a$,50)` = la chaîne entière.

Divergences assumées (l'interpréteur signale une erreur, le compilé continue) : `mid$(a$,0,…)` est traité
comme `mid$(a$,1,…)`, un argument négatif de `left$`/`right$`/`mid$`/`spc` vaut 0, `val(` d'un texte non
numérique vaut 0.

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
