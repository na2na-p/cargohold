#!/bin/bash

# =====================================
# E2Eテスト自動実行スクリプト
# =====================================
# このスクリプトは以下の処理を自動化します:
# 1. Docker Composeでテスト環境を起動
# 2. アプリケーションの起動を待機（readyzでDB/Redis/S3接続も確認）
# 3. E2Eテストを実行
# 4. テスト完了後、環境をクリーンアップ

set -e  # エラー時に即座に終了

# カラー出力の定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# スクリプトのディレクトリを取得
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Docker Composeファイルの設定（開発環境用）
export COMPOSE_FILE="compose.yml:compose.dev.yml"

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}E2Eテスト自動実行スクリプト${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""

# クリーンアップ関数
cleanup() {
    echo ""
    echo -e "${YELLOW}テスト環境をクリーンアップしています...${NC}"
    cd "${PROJECT_DIR}"
    docker compose down -v
    echo -e "${GREEN}クリーンアップが完了しました${NC}"
}

# トラップを設定してスクリプト終了時にクリーンアップを実行
trap cleanup EXIT INT TERM

# プロジェクトディレクトリに移動
cd "${PROJECT_DIR}"

# ステップ1: Docker Composeでテスト環境を起動
echo -e "${BLUE}[1/3] Docker Composeでテスト環境を起動しています...${NC}"
docker compose up -d --build

# ステップ2: アプリケーションの起動を待機（readyzでDB/Redis/S3接続も確認）
echo -e "${BLUE}[2/3] アプリケーションの起動を待機しています...${NC}"
echo -e "${YELLOW}  - readyzエンドポイントでDB/Redis/S3接続を確認中...${NC}"
elapsed=0
timeout=120
while ! curl -f http://localhost:8080/readyz > /dev/null 2>&1; do
    if [ $elapsed -ge $timeout ]; then
        echo -e "${RED}アプリケーションの起動がタイムアウトしました${NC}"
        echo -e "${YELLOW}コンテナログを確認してください:${NC}"
        docker compose logs
        exit 1
    fi
    sleep 2
    elapsed=$((elapsed + 2))
done
echo -e "${GREEN}  - アプリケーション起動完了（readyzチェック通過）${NC}"

# ステップ3: E2Eテストを実行（Docker内で実行）
echo ""
echo -e "${BLUE}[3/3] E2Eテストを実行しています...${NC}"
echo ""

# E2Eテストの実行（Dockerコンテナ内で実行）
if docker compose --profile e2e run --rm e2e-test; then
    echo ""
    echo -e "${GREEN}=====================================${NC}"
    echo -e "${GREEN}すべてのE2Eテストが成功しました！${NC}"
    echo -e "${GREEN}=====================================${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}=====================================${NC}"
    echo -e "${RED}E2Eテストが失敗しました${NC}"
    echo -e "${RED}=====================================${NC}"
    echo ""
    echo -e "${YELLOW}コンテナログを確認してください:${NC}"
    echo -e "${YELLOW}  docker compose logs${NC}"
    exit 1
fi
