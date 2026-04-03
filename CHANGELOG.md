# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-04-04

### Added
- **リソースダッシュボード**: Pod単位のCPU/メモリ利用率をテーブル表示、Namespace/Deploymentフィルタ、プログレスバー
- **コスト削減提案**: Request vs 実使用量の乖離分析、日本円ベース月間削減額計算、提案カード一覧
- **CronHPA生成**: 時間帯別負荷パターン分析、夜間/休日低負荷検出、CronJob YAMLテンプレート自動生成
- **Go APIサーバー**: chi router, graceful shutdown, CORS, PostgreSQL接続/マイグレーション
- **Kubernetes連携**: kubeconfig接続, Pod一覧/リソース取得, Metrics API連携
- **Next.jsダッシュボード**: 3ページ構成（リソース/提案/スケジュール）、サイドバーナビゲーション
- **CI/CD**: GitHub Actions (Go lint/test/build + Node lint/test/build)
- **テスト**: Backend 統合テスト含む全パッケージ、Frontend vitest 41テスト
- **ADR**: HTTPフレームワーク選定(chi)、コスト計算ロジック(GCPベース単価)
