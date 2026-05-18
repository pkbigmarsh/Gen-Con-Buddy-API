#!/bin/bash
set -euo pipefail

git config core.hooksPath .githooks
echo "✓ git hooks installed (core.hooksPath → .githooks)"
echo ""
echo "Ensure the following tools are installed:"
echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
echo "  go install golang.org/x/tools/cmd/goimports@latest"
