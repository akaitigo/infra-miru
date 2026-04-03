# Harvest: Infra-Miru

## プロジェクト概要
Kubernetesクラスタのリソース利用率可視化 + コスト最適化提案ダッシュボード。

## メトリクス

| Item | Value |
|------|-------|
| Commits | 7 |
| ADRs | 2 |
| CLAUDE.md lines | 38 |
| CI | YES |
| Issues (closed) | 5 |
| PRs (merged) | 5 |
| Go source files | 30 |
| TS/TSX source files | 16 |
| Backend test packages | 5 (analyzer, api, config, cost, k8s) |
| Frontend tests | 41 passed |
| Backend tests | 全パッケージ全パス |

## タイムライン
- Stage 1 (Launch): リポジトリ作成、スキャフォールド、5 Issue作成
- Stage 2 (Build): 5 Issue全完走、5 PR全マージ
- Stage 3 (Ship): README仕上げ、CHANGELOG、v1.0.0タグ
- Stage 4 (Harvest): 本ドキュメント

## テンプレート改善提案

### 良かった点
1. **Makefile テンプレート** — `check: lint test build` の統一インターフェースが全ステージで機能した
2. **ADRテンプレート** — Issue #1, #5 で不可逆な判断をADRに記録できた
3. **golangci-lint設定** — 厳格なlint設定がコード品質を維持

### 改善点
1. **golangci-lint v2 マイグレーション** — テンプレートの `.golangci.yml` がv1形式。v2形式に更新すべき
2. **biome v2 対応** — テンプレートの `biome.json` がv1スキーマ。v2に更新すべき
3. **モノレポMakefile** — フロントエンド+バックエンドのモノレポ用Makefileテンプレートがない。今回は手動で作成した
4. **Next.js 16対応** — create-next-appの出力が変わっており、テンプレートの `package.json.template` や `tsconfig.json` が古い可能性
5. **vitest setupFiles** — テスト環境のセットアップファイルパスをテンプレートに含めると初期設定が楽になる
6. **CI go-version-file** — テンプレートCIで `go-version` をハードコードするより `go-version-file: go.mod` が安定
