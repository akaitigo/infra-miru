# Infra-Miru — Agent Instructions

## アーキテクチャ
- モノレポ: frontend/ (Next.js) + backend/ (Go)
- backend は cmd/server/main.go がエントリポイント
- internal/ 以下にドメインロジック: analyzer, k8s, cost, api, config, db

## Go 規約
- エラーは必ず処理。`_` で握りつぶさない
- context.Context を第一引数に渡す
- テーブル駆動テストを使用
- golangci-lint (.golangci.yml) に従う

## TypeScript 規約
- any 禁止。unknown + 型ガードを使う
- oxlint + biome でlint/format
- vitest でテスト

## API設計
- RESTful JSON API
- エンドポイント: /api/v1/...
- エラーレスポンス: {"error": "message", "code": "ERROR_CODE"}

## 環境変数
- DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME — PostgreSQL接続
- KUBECONFIG — Kubernetes設定ファイルパス
- PROMETHEUS_URL — Prometheusエンドポイント
- PORT — APIサーバーポート (default: 8080)
