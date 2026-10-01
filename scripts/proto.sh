#!/usr/bin/env bash
set -euo pipefail

case "$#:${1:-}" in
0:) check=false ;;
1:--check) check=true ;;
*)
  echo "usage: $0 [--check]" >&2
  exit 2
  ;;
esac

cd "$(dirname "$0")/.."

if ! command -v buf >/dev/null 2>&1; then
  echo "buf not found: install it from https://buf.build/docs/installation" >&2
  exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

GOBIN="$work/bin" go install tool
export PATH="$work/bin:$PATH"

dirs=(gctrpc backtester/btrpc)
generated=(-name '*.pb.go' -o -name '*.pb.gw.go' -o -name '*.swagger.json')

for dir in "${dirs[@]}"; do
  mkdir -p "$work/tree/$dir"
  cp -RL "$dir/." "$work/tree/$dir"
  find "$work/tree/$dir" \( "${generated[@]}" \) -delete
  (cd "$work/tree/$dir" && buf generate)
done

if [ "$check" = true ]; then
  status=0
  for dir in "${dirs[@]}"; do
    diff -ru "$dir" "$work/tree/$dir" || status=1
  done
  if [ "$status" -ne 0 ]; then
    echo "Generated code is out of date: run make proto and commit the result" >&2
  fi
  exit "$status"
fi

for dir in "${dirs[@]}"; do
  find "$dir" \( "${generated[@]}" \) -delete
  (cd "$work/tree/$dir" && find . \( "${generated[@]}" \) -print0) |
    while IFS= read -r -d '' file; do
      mkdir -p "$(dirname "$dir/$file")"
      cp "$work/tree/$dir/$file" "$dir/$file"
    done
done
