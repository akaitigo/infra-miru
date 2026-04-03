#!/usr/bin/env bash
set -euo pipefail

echo "=== Session Startup ==="
[ -d ".git" ] || { echo "ERROR: Not in git repository"; exit 1; }

echo "=== Tool auto-install ==="
if [ -f "backend/go.mod" ]; then
  echo "Detected: Go"
  command -v golangci-lint &>/dev/null || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || echo "WARN: golangci-lint install failed"; }
  command -v gofumpt &>/dev/null || { echo "Installing gofumpt..."; go install mvdan.cc/gofumpt@latest 2>/dev/null || echo "WARN: gofumpt install failed"; }
fi
if [ -f "frontend/package.json" ]; then
  echo "Detected: TypeScript/JavaScript"
  command -v oxlint &>/dev/null || { echo "Installing oxlint..."; npm install -g oxlint 2>/dev/null || echo "WARN: oxlint install failed"; }
fi
command -v lefthook &>/dev/null || { echo "Installing lefthook..."; go install github.com/evilmartians/lefthook@latest 2>/dev/null || echo "WARN: lefthook install failed"; }
if command -v lefthook &>/dev/null && [ -f "lefthook.yml" ]; then
  lefthook install 2>/dev/null && echo "lefthook hooks installed." || echo "WARN: lefthook install failed"
fi
echo "Tool check complete."

echo "=== Recent commits ==="
git log --oneline -10

echo "=== Health check ==="
if make check 2>&1 | tail -10; then
  echo "All checks passed. Ready to work."
else
  echo "WARN: Checks failed. Review issues before proceeding."
fi

echo ""
echo "=== Session started at $(date -u +"%Y-%m-%dT%H:%M:%SZ") ==="
