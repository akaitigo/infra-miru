.PHONY: build test lint format check clean quality harvest

# === Build ===
build: build-backend build-frontend

build-backend:
	cd backend && go build -trimpath -ldflags "-s -w" ./...

build-frontend:
	cd frontend && npm run build

# === Test ===
test: test-backend test-frontend

test-backend:
	cd backend && go test -v -race -count=1 -coverprofile=coverage.out ./...

test-frontend:
	cd frontend && npx vitest run

# === Lint ===
lint: lint-backend lint-frontend

lint-backend:
	cd backend && golangci-lint run ./...

lint-frontend:
	cd frontend && npx oxlint .
	cd frontend && npx biome check .

# === Format ===
format: format-backend format-frontend

format-backend:
	cd backend && gofumpt -w .
	cd backend && goimports -w .

format-frontend:
	cd frontend && npx biome format --write .

# === Check (all) ===
check: lint test build
	@echo "All checks passed."

# === Quality ===
quality:
	@echo "=== Quality Gate ==="
	@test -f LICENSE || { echo "ERROR: LICENSE missing."; exit 1; }
	@! grep -rn "TODO\|FIXME\|HACK\|console\.log\|println\|print(" frontend/src/ backend/internal/ backend/cmd/ 2>/dev/null | grep -v "node_modules" || { echo "ERROR: debug output or TODO found."; exit 1; }
	@! grep -rn "password=\|secret=\|api_key=\|sk-\|ghp_" frontend/src/ backend/internal/ backend/cmd/ 2>/dev/null | grep -v '\$$' | grep -v "node_modules" || { echo "ERROR: hardcoded secrets."; exit 1; }
	@test ! -f CLAUDE.md || [ $$(wc -l < CLAUDE.md) -le 50 ] || { echo "ERROR: CLAUDE.md exceeds 50 lines."; exit 1; }
	@echo "OK: automated quality checks passed"

# === Clean ===
clean:
	cd backend && go clean -cache -testcache && rm -f coverage.out
	cd frontend && rm -rf dist/ .next/ node_modules/.cache/

# === Harvest ===
harvest:
	@echo "=== Harvest ==="
	@mkdir -p docs
	@echo "# Harvest: Infra-Miru" > docs/harvest.md
	@echo "" >> docs/harvest.md
	@echo "## Metrics" >> docs/harvest.md
	@echo "| Item | Value |" >> docs/harvest.md
	@echo "|------|-------|" >> docs/harvest.md
	@echo "| Commits | $$(git log --oneline --no-merges | wc -l) |" >> docs/harvest.md
	@echo "| ADRs | $$(ls docs/adr/*.md 2>/dev/null | wc -l) |" >> docs/harvest.md
	@echo "| CLAUDE.md lines | $$(wc -l < CLAUDE.md 2>/dev/null || echo 0) |" >> docs/harvest.md
	@echo "| CI | $$(test -f .github/workflows/ci.yml && echo YES || echo NO) |" >> docs/harvest.md
	@echo "" >> docs/harvest.md
	@echo "Harvest report generated: docs/harvest.md"
