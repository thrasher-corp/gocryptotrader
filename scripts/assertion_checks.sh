#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"
if ! command -v go >/dev/null 2>&1; then
    echo "Go is required for assertion checks; install the version from go.mod" >&2
    exit 1
fi

work_dir=$(mktemp -d)
trap 'rm -rf "$work_dir"' EXIT
git ls-files -z -- '*.go' > "$work_dir/tracked"
while IFS= read -r -d '' source; do
    printf '%s\0' "$repo_root/$source"
done < "$work_dir/tracked" > "$work_dir/sources"

go build -o "$work_dir/assertioncheck" ./cmd/assertioncheck
if [[ $# -eq 0 ]]; then
    set -- ./...
fi
go vet -vettool="$work_dir/assertioncheck" -assertionmessages.files="$work_dir/sources" "$@"
