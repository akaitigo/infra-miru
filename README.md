# Infra-Miru

Kubernetesクラスタのリソース利用率を可視化し、過剰プロビジョニングを検出してコスト最適化を提案するダッシュボード。

## Quick Start

```bash
# 1. 依存サービス起動
cp .env.example .env
# .env の DB_PASSWORD を設定
docker compose up -d

# 2. バックエンド
cd backend && go run ./cmd/server/

# 3. フロントエンド
cd frontend && npm install && npm run dev
```

## 技術スタック

| Layer | Technology |
|-------|-----------|
| Frontend | TypeScript / Next.js |
| Backend | Go |
| Database | PostgreSQL |
| Metrics | Prometheus / Grafana |

## 主要機能

1. **リソースダッシュボード** — Pod単位のCPU/メモリ利用率を可視化
2. **コスト削減提案** — Request vs 実使用量の乖離を日本円で換算
3. **CronHPA生成** — 夜間・休日の低負荷時間帯を自動検出し設定テンプレートを生成

> **Warning**: 本番利用前に認証・認可（RBAC統合）の実装が必要です。

## Development

```bash
make check    # lint + test + build
make lint     # lint only
make test     # test only
```

## License

MIT
