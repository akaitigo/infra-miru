# Infra-Miru

[![CI](https://github.com/akaitigo/infra-miru/actions/workflows/ci.yml/badge.svg)](https://github.com/akaitigo/infra-miru/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Kubernetesクラスタのリソース利用率を可視化し、過剰プロビジョニングを検出してコスト最適化を提案するダッシュボード。

## Demo

<!-- デモGIF/スクリーンショットをここに配置 -->
*Coming soon*

## 主要機能

1. **リソースダッシュボード** — Pod単位のCPU/メモリ利用率をリアルタイム可視化。Namespace/Deploymentフィルタ対応
2. **コスト削減提案** — Request vs 実使用量の乖離を分析し、日本円ベースの具体的な削減額を提示（例:「月¥8,000節約」）
3. **CronHPA生成** — 夜間(22:00-06:00)・休日の低負荷時間帯を自動検出し、スケールダウン用CronJob YAMLテンプレートを生成

## Quick Start

```bash
# 1. リポジトリクローン
git clone git@github.com:akaitigo/infra-miru.git
cd infra-miru

# 2. 環境変数設定
cp .env.example .env
# .env の DB_PASSWORD, KUBECONFIG を設定

# 3. PostgreSQL起動
docker compose up -d

# 4. バックエンドAPI起動
cd backend && go run ./cmd/server/

# 5. フロントエンド起動（別ターミナル）
cd frontend && npm install && npm run dev
```

ブラウザで http://localhost:3000 にアクセス。

## 技術スタック

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Frontend | TypeScript / Next.js 16 | ダッシュボードUI |
| Backend | Go 1.25 / chi | REST API / K8s連携 / 分析 |
| Database | PostgreSQL 16 | リソース履歴・提案DB |
| Metrics | Prometheus | メトリクス収集元 |

## アーキテクチャ

```
┌─────────────────┐     ┌──────────────────┐     ┌────────────┐
│   Next.js UI    │────▶│   Go API Server  │────▶│ Kubernetes │
│  (localhost:3000)│     │  (localhost:8080) │     │  Cluster   │
└─────────────────┘     └──────┬───────────┘     └────────────┘
                               │
                        ┌──────▼───────┐
                        │  PostgreSQL   │
                        │   (資源履歴)   │
                        └──────────────┘
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | ヘルスチェック |
| GET | `/api/v1/resources` | Pod一覧とリソース利用率 |
| GET | `/api/v1/recommendations` | コスト削減提案一覧 |
| GET | `/api/v1/schedules` | 時間帯別負荷パターン |
| GET | `/api/v1/cronhpa/{deployment}` | CronHPA YAMLテンプレート |

全エンドポイントは `?namespace=X&deployment=Y` でフィルタ可能。

## Development

```bash
make check    # format + lint + test + build（全チェック）
make lint     # lint only
make test     # test only
make build    # build only
make quality  # automated quality gate
```

## ADR (Architecture Decision Records)

- [ADR-0001: HTTPフレームワーク選定](docs/adr/0001-http-framework.md) — chi を選定
- [ADR-0002: コスト計算ロジック](docs/adr/0002-cost-calculation.md) — GCPベース単価

> **Warning**: 本番利用前に以下の実装が必要です:
> - 認証・認可（RBAC統合）
> - マルチクラスタ対応
> - カスタムコスト単価設定

## License

MIT
