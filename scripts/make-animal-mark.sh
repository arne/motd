#!/usr/bin/env bash
# Render an OpenMoji codepoint into a colorful ASCII-art mark.
# Usage: make-animal-mark.sh <name> <hex_codepoint> [out_dir]
set -euo pipefail

name=$1
cp=$2
out_dir=${3:-examples/marks/animals}
size=${SIZE:-24x14}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

mkdir -p "$out_dir"

curl -fsSL "https://github.com/hfg-gmuend/openmoji/raw/master/color/svg/${cp}.svg" -o "$tmp/src.svg"
convert -background none "$tmp/src.svg" -resize 512x512 "$tmp/hi.png"
convert "$tmp/hi.png" -trim +repage "$tmp/trim.png"
chafa --symbols=ascii --size "$size" --polite=off "$tmp/trim.png" \
  | sed -E 's/\x1b\[\?25[lh]//g' > "$out_dir/${name}.ansi"

printf 'wrote %s/%s.ansi\n' "$out_dir" "$name"
