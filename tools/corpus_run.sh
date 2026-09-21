#!/bin/sh
# Différentiel sur les exemples officiels : chaque .bsc est exécuté interprété et compilé dans
# Phosphoneo natif (--cycles, défaut 30 M) ; les écrans texte sont comparés (S9-1).
#   tools/corpus_run.sh [répertoire] [cycles]
set -u
dir=${1:-$HOME/Neo6502Trinity/examples}; cycles=${2:-30000000}
emu=${PHOSPHONEO:-$HOME/Phosphoneo}/build/phosphoneo
basic=${NEOBASIC_BIN:-$HOME/Neo6502Basic/bin/basic.bin}
go build -o neobas ./cmd/neobas && go build -o neoforgec ./cmd/neoforgec || exit 1
tmp=$(mktemp -d); same=0; diff=0; skip=0
for f in $(find "$dir" -name '*.bsc' | sort); do
  n=$(basename "$f" .bsc)
  if ! ./neoforgec -o "$tmp/p.neo" "$f" >/dev/null 2>&1; then skip=$((skip + 1)); continue; fi
  ./neobas -o "$tmp/p.bas" "$f" >/dev/null 2>&1
  rm -rf "$tmp/s1" "$tmp/s2"; mkdir -p "$tmp/s1" "$tmp/s2"
  cp "$(dirname "$f")"/*.gfx "$(dirname "$f")"/*.map "$(dirname "$f")"/*.dat "$tmp/s1" 2>/dev/null
  cp "$(dirname "$f")"/*.gfx "$(dirname "$f")"/*.map "$(dirname "$f")"/*.dat "$tmp/s2" 2>/dev/null
  "$emu" --headless --storage "$tmp/s1" "$basic@800" "$tmp/p.bas" --cycles "$cycles" --screenshot-text "$tmp/i.txt" >/dev/null 2>&1
  "$emu" --headless --storage "$tmp/s2" "$tmp/p.neo" --cycles "$cycles" --screenshot-text "$tmp/c.txt" >/dev/null 2>&1
  if cmp -s "$tmp/i.txt" "$tmp/c.txt"; then same=$((same + 1)); printf '%-24s identique\n' "$n"
  else diff=$((diff + 1)); printf '%-24s DIFFÉRENT\n' "$n"; mkdir -p "$tmp/keep"; cp "$tmp/i.txt" "$tmp/keep/$n.i.txt"; cp "$tmp/c.txt" "$tmp/keep/$n.c.txt"; fi
done
echo "identiques : $same, différents : $diff, non compilés : $skip (écrans : $tmp/keep)"
