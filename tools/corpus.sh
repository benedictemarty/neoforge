#!/bin/sh
# Mesure du taux de compilation sur les exemples NeoBASIC du firmware (S4-5).
#   tools/corpus.sh [répertoire]   (défaut : ~/Neo6502Trinity/examples)
set -u
dir=${1:-$HOME/Neo6502Trinity/examples}
go build -o neoforgec ./cmd/neoforgec || exit 1
ok=0; total=0; tmp=$(mktemp -d)
for f in $(find "$dir" -name '*.bsc' | sort); do
  total=$((total + 1))
  if ./neoforgec -o "$tmp/x.neo" "$f" >/dev/null 2>"$tmp/err"; then ok=$((ok + 1))
  else printf '%-28s %s\n' "$(basename "$f")" "$(head -1 "$tmp/err")"; fi
done
rm -rf "$tmp"
echo "compilés : $ok / $total"
