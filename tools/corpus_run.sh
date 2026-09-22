#!/bin/bash
# Différentiel sur les exemples officiels : chaque .bsc est exécuté interprété et compilé dans
# Phosphoneo natif, les écrans texte sont comparés (S9-1, S10-1).
#   tools/corpus_run.sh [répertoire] [cycles]
#
# Un écart peut être légitime : adresses d'`alloc(` (le tas ne commence pas au même endroit) et
# chronométrages (`time()` — c'est l'objet des benchmarks). Une comparaison qui échoue est donc
# rejouée avec un budget de cycles plus large (les deux exécutions terminent), puis sous deux
# normalisations successives : adresses hexadécimales en tête de ligne, puis nombres décimaux.
# Les programmes sans fin (boucle `until 0`) ne sont comparables que sur leur préfixe commun.
set -u
dir=${1:-$HOME/Neo6502Trinity/examples}; cycles=${2:-30000000}; long=${3:-2000000000}
emu=${PHOSPHONEO:-$HOME/Phosphoneo}/build/phosphoneo
basic=${NEOBASIC_BIN:-$HOME/Neo6502Basic/bin/basic.bin}
go build -o neobas ./cmd/neobas && go build -o neoforgec ./cmd/neoforgec || exit 1
tmp=$(mktemp -d); same=0; addr=0; chrono=0; prefix=0; diffs=0; skip=0
clean() { grep -v '^[[:space:]]*$' "$1" | sed -e 's/[[:space:]]*$//'; }

run_pair() { # $1 = cycles : joue le .bas interprété et le .neo compilé
  rm -rf "$tmp/s1" "$tmp/s2"; mkdir -p "$tmp/s1" "$tmp/s2"
  for ext in gfx map dat; do cp "$(dirname "$f")"/*.$ext "$tmp/s1" 2>/dev/null; cp "$(dirname "$f")"/*.$ext "$tmp/s2" 2>/dev/null; done
  "$emu" --headless --storage "$tmp/s1" "$basic@800" "$tmp/p.bas" --cycles "$1" --screenshot-text "$tmp/i.txt" >/dev/null 2>&1
  "$emu" --headless --storage "$tmp/s2" "$tmp/p.neo" --cycles "$1" --screenshot-text "$tmp/c.txt" >/dev/null 2>&1
}

for f in $(find "$dir" -name '*.bsc' | sort); do
  n=$(basename "$f" .bsc)
  if ! ./neoforgec -o "$tmp/p.neo" "$f" >/dev/null 2>&1; then skip=$((skip + 1)); continue; fi
  ./neobas -o "$tmp/p.bas" "$f" >/dev/null 2>&1
  run_pair "$cycles"
  if cmp -s "$tmp/i.txt" "$tmp/c.txt"; then same=$((same + 1)); printf '%-24s identique\n' "$n"; continue; fi
  cp "$tmp/i.txt" "$tmp/i0.txt"; cp "$tmp/c.txt" "$tmp/c0.txt" # écrans du budget court (avant défilement)
  run_pair "$long" # les deux exécutions vont au bout
  if diff -q <(clean "$tmp/i.txt" | sed -E 's/^[0-9A-F]{4} //') <(clean "$tmp/c.txt" | sed -E 's/^[0-9A-F]{4} //') >/dev/null; then
    addr=$((addr + 1)); printf '%-24s identique (adresses alloc( masquées)\n' "$n"; continue
  fi
  if diff -q <(clean "$tmp/i.txt" | sed -E 's/[0-9]+\.[0-9]+/T/g') <(clean "$tmp/c.txt" | sed -E 's/[0-9]+\.[0-9]+/T/g') >/dev/null; then
    chrono=$((chrono + 1))
    printf '%-24s identique (chronos masqués) : interprété %s / compilé %s\n' "$n" \
      "$(clean "$tmp/i.txt" | grep -oE '[0-9]+\.[0-9]+' | tail -1)" "$(clean "$tmp/c.txt" | grep -oE '[0-9]+\.[0-9]+' | tail -1)"
    continue
  fi
  # Programme sans fin (`until 0`) : au budget court, l'exécution compilée a produit plus de lignes
  # que l'interprétée mais les premières doivent coïncider (même suite pseudo-aléatoire depuis la
  # remise à zéro). Au budget long, l'écran a défilé des deux côtés : il n'est plus comparable.
  ni=$(clean "$tmp/i0.txt" | wc -l); nc=$(clean "$tmp/c0.txt" | wc -l); k=$((ni < nc ? ni : nc))
  if [ "$k" -gt 3 ] && diff -q <(clean "$tmp/i0.txt" | head -n "$k") <(clean "$tmp/c0.txt" | head -n "$k") >/dev/null; then
    prefix=$((prefix + 1)); printf '%-24s identique (préfixe %d lignes ; sans fin : %d lignes compilé contre %d)\n' "$n" "$k" "$nc" "$ni"; continue
  fi
  diffs=$((diffs + 1)); printf '%-24s DIFFÉRENT\n' "$n"
  mkdir -p "$tmp/keep"; cp "$tmp/i.txt" "$tmp/keep/$n.i.txt"; cp "$tmp/c.txt" "$tmp/keep/$n.c.txt"
done
echo "identiques : $same, adresses : $addr, chronos : $chrono, préfixes : $prefix, différents : $diffs, non compilés : $skip"
[ "$diffs" -gt 0 ] && echo "écrans conservés : $tmp/keep"
exit 0
