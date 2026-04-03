# ADR-0001: HTTPフレームワーク選定

## ステータス

Accepted

## コンテキスト

Infra-MiruのバックエンドAPIサーバーに使用するHTTPルーター/フレームワークを選定する必要がある。
候補は以下の3つ。

| 候補       | 特徴                                            |
| ---------- | ----------------------------------------------- |
| net/http   | 標準ライブラリ。Go 1.22でルーティング改善済み    |
| chi        | net/http互換の軽量ルーター。ミドルウェア豊富     |
| echo       | 独自のContext型。高機能だがnet/httpとの互換性低い |

## 決定

**chi (go-chi/chi/v5)** を採用する。

## 理由

1. **net/http互換**: `http.Handler` / `http.HandlerFunc` をそのまま使える。標準ライブラリのテストツール（httptest）がそのまま利用可能
2. **ミドルウェアエコシステム**: RequestID、Logger、Recoverer、CORS、Timeout等が公式パッケージで提供されている
3. **軽量**: 依存が最小限。バイナリサイズへの影響が小さい
4. **context活用**: Go標準の `context.Context` をそのまま使用。独自のContext型を導入しない
5. **学習コスト**: net/httpの知識がそのまま適用できるため、チームの立ち上がりが速い

### 不採用理由

- **net/http単体**: Go 1.22でルーティングが改善されたが、ミドルウェアチェーンの組み立てやグループ化にボイラープレートが必要
- **echo**: 独自のContext型（`echo.Context`）がnet/httpの標準インターフェースと互換性がない。テストで`httptest`を直接使えず、フレームワーク依存が強くなる

## 影響

- 全HTTPハンドラーは `http.HandlerFunc` として実装する
- ミドルウェアは `func(http.Handler) http.Handler` シグネチャに従う
- テストは `net/http/httptest` で記述する
- フレームワーク乗り換え時の影響が最小限に抑えられる
