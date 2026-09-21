#!/bin/sh
# Mesure interprété / compilé de tools/bench/bench.bsc dans Phosphoneo natif (S8-1).
#   tools/bench.sh          → temps en 1/100 s (time()) des cinq boucles, puis le total
set -u
emu=${PHOSPHONEO:-$HOME/Phosphoneo}/build/phosphoneo
basic=${NEOBASIC_BIN:-$HOME/Neo6502Basic/bin/basic.bin}
go build -o neobas ./cmd/neobas && go build -o neoforgec ./cmd/neoforgec || exit 1
tmp=$(mktemp -d); mkdir -p "$tmp/storage/boot"
cp "$basic" "$tmp/storage/boot/neobasic.bin" && echo neobasic.bin > "$tmp/storage/boot/auto.txt"
./neobas -o "$tmp/bench.bas" tools/bench/bench.bsc >/dev/null && ./neoforgec -o "$tmp/bench.neo" tools/bench/bench.bsc >/dev/null || exit 1
echo "== interprété"
"$emu" --headless --storage "$tmp/storage" "$basic@800" "$tmp/bench.bas" --cycles 800000000 --screenshot-text "$tmp/i.txt" >/dev/null 2>&1
grep -v '^\s*$' "$tmp/i.txt" | tail -6
echo "== compilé"
"$emu" --headless --storage "$tmp/storage" "$tmp/bench.neo" --cycles 800000000 --screenshot-text "$tmp/c.txt" >/dev/null 2>&1
grep -v '^\s*$' "$tmp/c.txt" | tail -6
rm -rf "$tmp"
