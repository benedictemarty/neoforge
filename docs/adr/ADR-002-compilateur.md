# ADR-002 — Conception du compilateur NeoBASIC → 65C02

Date : 2026-09-20 · État : **accepté** (sprint 2 ouvert)

## Contexte

Le compilateur d'oriced (≈ 11 000 lignes hors tests) est intimement lié à l'Oric : AST du BASIC Oric
(numéros de ligne, `GOTO`), accumulateur entier 16 bits en page zéro, flottant par les routines ROM,
écran `$BB80`. NeoBASIC est un autre langage : entiers **32 bits signés** et flottants IEEE simple
précision (valeur de 4 octets + octet de type, cf. `reference/api.md` §Mathematics), structures
(`while/wend`, `repeat/until`, `do/loop`, `if/endif`, `proc/endproc` avec paramètres et `local`),
commandes graphiques chaînées (`line … ink … to …`), pas de numéro de ligne obligatoire. Le matériel est
atteint uniquement par l'**API `$FF00`** (groupes 1-7 : système, console, fichiers, maths, graphisme,
sprites, manette) : `SendMessage`/`WaitMessage` du noyau (`documents/release/neo6502.inc`).

## Décisions

1. **Nouveau compilateur** (`internal/compiler`), pas un portage ligne à ligne. Repris d'oriced : le
   modèle de l'assembleur (`asm.go` : étiquettes, rétro-correction, relaxation de branches, élagage du
   runtime mort), l'architecture « frontend → codegen → runtime émis après le corps », la discipline
   (100 % de couverture, chaque feature validée sur l'émulateur, harnais différentiel interprété/compilé).
2. **Frontend sur le flux de tokens** (`internal/neobasic`) et non sur le texte : mêmes mots-clés,
   mêmes priorités (modificateurs `:1`…`:4` de la table), mêmes constantes que l'interpréteur — la
   sémantique lexicale est alignée par construction.
3. **Assembleur 65C02** intégré (`BRA`, `STZ`, `PHX/PHY/PLX/PLY`, `INC A/DEC A`, `TRB/TSB`, `(zp)`),
   validé en différentiel contre **64tass** (oracle) sur des listings générés.
4. **Runtime sur l'API `$FF00`** : macro d'appel (écrire fonction/groupe en `$FF01/$FF00`, attendre
   `$FF00 = 0`), paramètres `$FF04..$FF0B`. Arithmétique : entiers 32 bits **en 6502 natif** (add/sub/
   comparaisons/décalages ; mul/div/mod par l'API 4,2/4,4/4,5), flottants et fonctions (`sin`, `sqr`,
   `str$`, `val`…) par le **groupe 4** avec des registres au format de l'interpréteur. Console : 2,6 (`print`),
   2,1 (`inkey$`), 2,3 (`input`), 2,12 (`cls`). Graphisme/sprites : groupes 5-6, mêmes paramètres que
   `gfxcommands.cpp`.
5. **Sortie** : binaire chargé en `$800` (comme un `.bin` du menu `boot/`) ou emballé en **`.neo`**
   (format de `fileinterface.cpp`, cf. `~/Phosphoneo/tools/mkneo.py` : `03 'N' 'E' 'O' 02 00 exec_lo
   exec_hi`, blocs `[more][adr][len][0][données]`, bloc `$FFFF` = mémoire graphique) — exécutable dans
   Phosphoneo (`prog.neo`), sur la carte (`load` NeoBASIC / menu `boot/`).
6. **Oracle de fidélité** : pour chaque programme du corpus, sortie de l'interpréteur (`.bas` +
   `--screenshot-text`) = sortie du compilé (`.neo` + `--screenshot-text`), comme la campagne E7 d'oriced.

## Découpage (sprint 2 = fondations)

| Récit | Contenu |
|---|---|
| S2-2 | Assembleur 65C02 + oracle 64tass |
| S2-4 | Emballage `.neo`, exécution dans Phosphoneo, `neoforgec` CLI |
| S2-1 | Parseur : expressions (priorités de la table), `let`, `print`, `if/endif`, boucles, `proc` |
| S2-3 | Runtime : appel API, entiers 32 bits, chaînes, `print`, boucles ; premier programme compilé |
| S2-5 | Harnais différentiel interprété/compilé |

Hors sprint 2 : flottants, graphisme/sprites, fichiers, son, assembleur en ligne `[ ]`, `goto/gosub`.
