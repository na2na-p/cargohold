# cargohold

## プロジェクト概要

cargohold は、Git LFS (Git Large File Storage) の自前実装サーバーです。

Git LFS は、大容量ファイル（動画、画像、モデルデータなど）を Git で扱う際の課題を解決するための標準仕様です。cargohold は、Git LFS Batch API を提供し、S3 互換ストレージ、PostgreSQL、Redis を統合することで、スケーラブルで高性能な LFS サーバーを実現します。

### 主な機能

- **Git LFS Batch API 実装**: `download` および `upload` オペレーションに対応
- **S3 互換ストレージ統合**: 署名付き URL による直接アップロード・ダウンロード
- **PostgreSQL 統合**: LFS オブジェクトのメタデータ管理
- **Redis キャッシュ統合**: レイテンシーを低減するキャッシュ機構
- **マルチ認証**: GitHub OIDC（CI/CD）、GitHub OAuth、セッション認証をサポート
- **DDD アーキテクチャ**: Handler → UseCase → Domain → Infrastructure の層構造

## 前提条件

以下のツールがインストールされている必要があります：

- **Go**: 1.25 以降
- **Docker**: 20.10 以降
- **Docker Compose**: 2.0 以降
- **Git**: 2.30 以降（Git LFS クライアントをテストする場合）
- **Git LFS**: 3.0 以降（クライアント側のテストを行う場合）

### 動作確認

```bash
# Goのバージョン確認
go version

# Dockerのバージョン確認
docker --version
docker compose version

# Git LFSのバージョン確認（オプション）
git lfs version
```

## マイグレーションツールのインストール

cargohold では、データベーススキーマのバージョン管理に [golang-migrate](https://github.com/golang-migrate/migrate) を使用しています。

### CLI ツールのインストール

#### macOS

```bash
brew install golang-migrate
```

#### Linux

```bash
curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/migrate
```

#### Go install

```bash
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### マイグレーションの実行

```bash
# マイグレーションを適用
./scripts/migrate.sh up

# マイグレーションを巻き戻し
./scripts/migrate.sh down

# 現在のマイグレーションバージョンを確認
./scripts/migrate.sh version
```

## 環境構築手順

### 1. リポジトリのクローン

```bash
# リポジトリをクローン
git clone <repository-url>
cd cargohold
```

### 2. 依存ライブラリのインストール

```bash
# Goモジュールのダウンロード
go mod download

# 依存関係の確認
go mod verify
```

### 3. 環境変数の設定

開発環境用の環境変数テンプレートをコピーします。

```bash
# .env.exampleを.envにコピー
cp .env.example .env
```

必要に応じて `.env` ファイルの値を編集してください。

### 4. 開発環境の起動

Docker Compose を使用して、PostgreSQL、Redis、SeaweedFS を起動します。

```bash
# バックグラウンドで起動（開発環境）
docker compose -f compose.yml -f compose.dev.yml up -d

# ログを確認する場合
docker compose -f compose.yml -f compose.dev.yml up

# 起動状態の確認
docker compose -f compose.yml -f compose.dev.yml ps
```

### 5. ヘルスチェック確認

各サービスが正常に起動しているか確認します。

```bash
# PostgreSQLのヘルスチェック
docker exec cargohold-postgres pg_isready -U cargohold

# Redisのヘルスチェック
docker exec cargohold-redis redis-cli ping

# SeaweedFSのヘルスチェック
curl -f http://localhost:9333/cluster/status
```

すべてのサービスが正常であれば、以下のような出力が得られます：

- PostgreSQL: `cargohold-postgres:5432 - accepting connections`
- Redis: `PONG`
- SeaweedFS: `200 OK`

## 開発環境の起動方法

### Docker Compose による開発環境起動

```bash
# すべてのサービスを起動
docker compose -f compose.yml -f compose.dev.yml up -d

# 特定のサービスのみ起動
docker compose -f compose.yml -f compose.dev.yml up -d postgres redis

# ログの確認
docker compose -f compose.yml -f compose.dev.yml logs -f

# サービスの停止
docker compose -f compose.yml -f compose.dev.yml down

# データを含めて完全に削除
docker compose -f compose.yml -f compose.dev.yml down -v
```

### 各サービスのアクセス情報

#### PostgreSQL

- **ホスト**: `localhost`
- **ポート**: `5432`
- **ユーザー名**: `cargohold`
- **パスワード**: `cargohold_dev_password`
- **データベース名**: `cargohold`

接続例：

```bash
# psqlで接続
docker exec -it cargohold-postgres psql -U cargohold -d cargohold

# または、ホストから接続
psql -h localhost -p 5432 -U cargohold -d cargohold
```

#### Redis

- **ホスト**: `localhost`
- **ポート**: `6379`
- **パスワード**: なし（開発環境）

接続例：

```bash
# redis-cliで接続
docker exec -it cargohold-redis redis-cli

# または、ホストから接続
redis-cli -h localhost -p 6379
```

#### SeaweedFS (S3 互換ストレージ)

- **S3 API エンドポイント**: `http://localhost:9000`
- **Master API**: `http://localhost:9333`
- **アクセスキー**: `minioadmin`
- **シークレットキー**: `minioadmin_dev_password`

SeaweedFS は軽量な S3 互換オブジェクトストレージです。S3 API を通じてアクセスできます。

### cargohold サーバーの起動

```bash
# サーバーを起動
go run cmd/cargohold/main.go

# または、ビルドしてから実行
go build -o bin/cargohold cmd/cargohold/main.go
./bin/cargohold
```

サーバーが起動すると、デフォルトで `http://localhost:8080` でリッスンします。

### ヘルスチェック確認方法

サーバーが起動したら、以下のコマンドでヘルスチェックを確認できます：

```bash
# ヘルスチェックエンドポイント
curl http://localhost:8080/healthz

# Readinessエンドポイント
curl http://localhost:8080/readyz

# Batch APIエンドポイントの確認
# URL形式: /{owner}/{repo}/info/lfs/objects/batch
curl -X POST http://localhost:8080/na2na-p/test-repo/info/lfs/objects/batch \
	-H "Accept: application/vnd.git-lfs+json" \
	-H "Content-Type: application/vnd.git-lfs+json" \
	-H "Authorization: Basic $(echo -n 'admin:admin_password' | base64)" \
	-d '{
		"operation": "download",
		"transfers": ["basic"],
		"objects": [
			{
				"oid": "test",
				"size": 0
			}
		]
	}'
```

## ディレクトリ構造

cargohold は、DDD アーキテクチャに基づいた以下のディレクトリ構成を採用しています：

```
cargohold/
├── cmd/
│   ├── cargohold/
│   │   └── main.go              # アプリケーションのエントリーポイント
│   └── generate-test-keys/      # テスト用キー生成ツール
├── internal/
│   ├── config/                  # 設定管理（環境変数、設定ファイル読み込み）
│   ├── domain/                  # ドメイン層（エンティティ、値オブジェクト、Repositoryインターフェース）
│   ├── usecase/                 # UseCase層（ビジネスロジック）
│   ├── infrastructure/          # Infrastructure層（Repository実装、外部サービス連携）
│   │   ├── postgres/            # PostgreSQL DAO・Repository実装
│   │   ├── redis/               # Redis Client・キャッシュ実装
│   │   ├── s3/                  # S3 Client・署名付きURL生成
│   │   ├── oidc/                # OIDC Provider実装（GitHub）
│   │   └── logging/             # ロギング
│   └── handler/                 # Handler層（HTTPハンドラー、ルーティング）
│       ├── dto/                 # Data Transfer Objects
│       ├── middleware/          # ミドルウェア（認証、エラーハンドリング）
│       ├── response/            # レスポンス構造体
│       ├── auth/                # 認証関連ハンドラー
│       └── common/              # 共通ユーティリティ
├── migrations/                  # データベースマイグレーションファイル
├── config/                      # 設定ファイル（SeaweedFS等）
├── tests/                       # テストモック等
├── e2e/                         # E2Eテスト
├── examples/                    # サンプルファイル
├── scripts/                     # 開発・運用スクリプト
├── tasks/                       # タスク定義
├── .github/                     # GitHub Actions設定
├── compose.yml                  # Docker Compose ベース設定
├── compose.dev.yml              # Docker Compose 開発環境用オーバーライド
├── compose.ci.yml               # Docker Compose CI環境用設定
├── Dockerfile                   # アプリケーション用Dockerfile
├── Dockerfile.e2e               # E2Eテスト用Dockerfile
├── .env.example                 # 環境変数テンプレート
├── go.mod                       # Goモジュール定義
└── README.md                    # このファイル
```

### 層の役割

#### Handler 層 (`internal/handler/`)

HTTP リクエストの受付、バリデーション、レスポンス生成を担当します。

- `POST /{owner}/{repo}/info/lfs/objects/batch`: Git LFS Batch API
- `POST /{owner}/{repo}/info/lfs/objects/verify`: アップロード完了通知
- `GET /healthz`: ヘルスチェック
- `GET /readyz`: Readiness チェック

#### UseCase 層 (`internal/usecase/`)

ビジネスロジックのオーケストレーションを担当します。

- `BatchDownloadUseCase`: ダウンロードオペレーション
- `BatchUploadUseCase`: アップロードオペレーション
- `AuthUseCase`: 認証処理（Basic、セッション）
- `GitHubOIDCUseCase`: GitHub OIDC 認証
- `GitHubOAuthUseCase`: GitHub OAuth 認証
- `VerifyUseCase`: アップロード完了通知処理

#### Domain 層 (`internal/domain/`)

ドメインモデル、値オブジェクト、Repository インターフェースを定義します。

- `LFSObject`: LFS オブジェクトのエンティティ
- `OID`, `Size`, `StorageKey`: 値オブジェクト
- `Repository`: リポジトリインターフェース
- `AccessPolicy`: アクセスポリシー
- `AccessAuthorizationService`: 認可サービス

#### Infrastructure 層 (`internal/infrastructure/`)

Repository インターフェースの実装と外部サービスとの連携を提供します。

- `postgres/`: PostgreSQL との通信
- `redis/`: Redis との通信・キャッシュ
- `s3/`: S3 互換ストレージとの通信
- `oidc/`: OIDC 認証プロバイダー実装（GitHub）
- `logging/`: ロギング機能

## E2Eテスト

E2Eテストを実行するには：

```bash
# E2Eテストの実行
./scripts/e2e_test.sh

# または、Docker Composeで実行
docker compose -f compose.yml -f compose.dev.yml --profile e2e run --rm e2e-test
```

## 参考資料

### 外部リンク

- [Git LFS Specification](https://github.com/git-lfs/git-lfs/tree/main/docs/api): Git LFS 公式仕様
- [Git LFS Batch API](https://github.com/git-lfs/git-lfs/blob/main/docs/api/batch.md): Batch API 仕様

## トラブルシューティング

### Docker Compose が起動しない

```bash
# コンテナの状態確認
docker compose -f compose.yml -f compose.dev.yml ps

# ログの確認
docker compose -f compose.yml -f compose.dev.yml logs

# コンテナの再起動
docker compose -f compose.yml -f compose.dev.yml restart

# 完全にクリーンアップして再起動
docker compose -f compose.yml -f compose.dev.yml down -v
docker compose -f compose.yml -f compose.dev.yml up -d
```

### PostgreSQL 接続エラー

```bash
# PostgreSQLの起動確認
docker exec cargohold-postgres pg_isready -U cargohold

# ログの確認
docker compose -f compose.yml -f compose.dev.yml logs postgres

# データベースの再作成
docker compose -f compose.yml -f compose.dev.yml down -v
docker compose -f compose.yml -f compose.dev.yml up -d postgres
```

### Redis 接続エラー

```bash
# Redisの起動確認
docker exec cargohold-redis redis-cli ping

# ログの確認
docker compose -f compose.yml -f compose.dev.yml logs redis
```

### SeaweedFS 接続エラー

```bash
# SeaweedFSの起動確認
curl http://localhost:9333/cluster/status

# S3 APIエンドポイントの確認
curl http://localhost:9000/status

# ログの確認
docker compose -f compose.yml -f compose.dev.yml logs seaweedfs
```

## ライセンス

（ライセンス情報を追加してください）
