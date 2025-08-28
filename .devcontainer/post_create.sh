#!/usr/bin/env bash
set -euo pipefail

command -v go >/dev/null 2>&1 || { echo "Go is not installed" >&2; exit 1; }

log() { echo "[post-create] $*"; }

install_go_tool() {
  local pkg="$1"
  log "Installing: $pkg"
  go install "$pkg"
}

# Protobuf + gRPC codegen
install_go_tool google.golang.org/protobuf/cmd/protoc-gen-go@latest
install_go_tool google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
install_go_tool github.com/bufbuild/buf/cmd/buf@latest
install_go_tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
install_go_tool github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

# Linting
install_go_tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0

# Formatting
install_go_tool mvdan.cc/gofumpt@latest

# Go modernise enforcer
install_go_tool golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest

log "Installed tools:"

for t in protoc-gen-go protoc-gen-go-grpc buf protoc-gen-grpc-gateway protoc-gen-openapiv2 golangci-lint gofumpt modernize; do
  command -v "$t" || true
done

log "Setup is complete :)"
