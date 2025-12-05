#!/bin/bash
# ==============================================
# Git LFS セットアップスクリプト
# ==============================================
# cargohold サーバーを使用するための Git LFS 初期設定を行います。
#
# 使い方:
#   ./git-lfs-setup.sh [cargohold_url]
#
# 例:
#   ./git-lfs-setup.sh https://cargohold.example.com/info/lfs
#   ./git-lfs-setup.sh  # デフォルト URL を使用

set -e

# ==============================================
# 設定
# ==============================================

# デフォルトの cargohold サーバー URL
DEFAULT_LFS_URL="https://cargohold.example.com/info/lfs"

# 引数から URL を取得、なければデフォルトを使用
LFS_URL="${1:-$DEFAULT_LFS_URL}"

# カラー出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ==============================================
# 関数定義
# ==============================================

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        error "$1 がインストールされていません。インストールしてから再実行してください。"
    fi
}

# ==============================================
# 前提条件チェック
# ==============================================

info "前提条件をチェックしています..."

# Git がインストールされているか確認
check_command "git"
GIT_VERSION=$(git --version | awk '{print $3}')
info "Git バージョン: $GIT_VERSION"

# Git LFS がインストールされているか確認
check_command "git-lfs"
GIT_LFS_VERSION=$(git lfs version | head -1 | awk -F'/' '{print $2}' | awk '{print $1}')
info "Git LFS バージョン: $GIT_LFS_VERSION"

# Git リポジトリ内にいるか確認
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "Git リポジトリ内で実行してください。"
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
info "リポジトリルート: $REPO_ROOT"

# ==============================================
# Git LFS セットアップ
# ==============================================

info "Git LFS をセットアップしています..."

# Git LFS をインストール（グローバル設定）
git lfs install
success "Git LFS がインストールされました。"

# ==============================================
# .lfsconfig 設定
# ==============================================

LFSCONFIG_FILE="$REPO_ROOT/.lfsconfig"

if [ -f "$LFSCONFIG_FILE" ]; then
    warn ".lfsconfig は既に存在します。上書きをスキップします。"
else
    info "cargohold サーバーを設定しています..."
    cat > "$LFSCONFIG_FILE" << EOF
[lfs]
    url = $LFS_URL
EOF
    success ".lfsconfig ファイルを作成しました。"
fi

# ==============================================
# .gitattributes 設定（サンプル）
# ==============================================

GITATTRIBUTES_FILE="$REPO_ROOT/.gitattributes"

if [ ! -f "$GITATTRIBUTES_FILE" ]; then
    info ".gitattributes ファイルを作成しています..."

    cat > "$GITATTRIBUTES_FILE" << 'EOF'
# ==============================================
# Git LFS 追跡パターン
# ==============================================
# 大容量ファイルを Git LFS で管理するためのパターン定義
#
# パターンの追加方法:
#   git lfs track "*.extension"
#
# 参考: https://git-lfs.com/

# 画像ファイル
*.png filter=lfs diff=lfs merge=lfs -text
*.jpg filter=lfs diff=lfs merge=lfs -text
*.jpeg filter=lfs diff=lfs merge=lfs -text
*.gif filter=lfs diff=lfs merge=lfs -text
*.bmp filter=lfs diff=lfs merge=lfs -text
*.tiff filter=lfs diff=lfs merge=lfs -text
*.ico filter=lfs diff=lfs merge=lfs -text
*.webp filter=lfs diff=lfs merge=lfs -text

# 動画ファイル
*.mp4 filter=lfs diff=lfs merge=lfs -text
*.mov filter=lfs diff=lfs merge=lfs -text
*.avi filter=lfs diff=lfs merge=lfs -text
*.mkv filter=lfs diff=lfs merge=lfs -text
*.webm filter=lfs diff=lfs merge=lfs -text

# 音声ファイル
*.mp3 filter=lfs diff=lfs merge=lfs -text
*.wav filter=lfs diff=lfs merge=lfs -text
*.ogg filter=lfs diff=lfs merge=lfs -text
*.flac filter=lfs diff=lfs merge=lfs -text

# アーカイブファイル
*.zip filter=lfs diff=lfs merge=lfs -text
*.tar filter=lfs diff=lfs merge=lfs -text
*.tar.gz filter=lfs diff=lfs merge=lfs -text
*.tgz filter=lfs diff=lfs merge=lfs -text
*.rar filter=lfs diff=lfs merge=lfs -text
*.7z filter=lfs diff=lfs merge=lfs -text

# ドキュメントファイル
*.pdf filter=lfs diff=lfs merge=lfs -text
*.psd filter=lfs diff=lfs merge=lfs -text
*.ai filter=lfs diff=lfs merge=lfs -text

# 機械学習モデル
*.h5 filter=lfs diff=lfs merge=lfs -text
*.hdf5 filter=lfs diff=lfs merge=lfs -text
*.pb filter=lfs diff=lfs merge=lfs -text
*.onnx filter=lfs diff=lfs merge=lfs -text
*.pt filter=lfs diff=lfs merge=lfs -text
*.pth filter=lfs diff=lfs merge=lfs -text
*.pkl filter=lfs diff=lfs merge=lfs -text
*.model filter=lfs diff=lfs merge=lfs -text

# その他のバイナリファイル
*.exe filter=lfs diff=lfs merge=lfs -text
*.dll filter=lfs diff=lfs merge=lfs -text
*.so filter=lfs diff=lfs merge=lfs -text
*.dylib filter=lfs diff=lfs merge=lfs -text
EOF

    success ".gitattributes ファイルを作成しました。"
else
    warn ".gitattributes ファイルは既に存在します。必要に応じて手動で更新してください。"
fi

# ==============================================
# ベースURL導出
# ==============================================
if [[ "$LFS_URL" =~ ^(.+)/info/lfs(/.*)?$ ]]; then
    CARGOHOLD_SERVER="${BASH_REMATCH[1]}"
else
    warn "LFS_URL が期待されるパターン (/info/lfs) で終わっていません: $LFS_URL"
    warn "認証URLが正しくない可能性があります。"
    CARGOHOLD_SERVER="$LFS_URL"
fi

# ==============================================
# 認証設定
# ==============================================

echo ""
info "認証設定について:"
echo ""
echo "  cargohold は以下の認証方式をサポートしています："
echo ""
echo "  1. Basic 認証"
echo "     git config --global credential.helper store"
echo "     # または"
echo "     git config --global credential.helper 'cache --timeout=3600'"
echo ""
echo "  2. Google Workspace OIDC（ブラウザ認証）"
echo "     ブラウザで ${CARGOHOLD_SERVER}/auth/google/login にアクセス"
echo ""
echo "  3. GitHub OIDC（CI/CD 向け）"
echo "     GitHub Actions から自動的に認証されます。"
echo ""

# ==============================================
# 設定確認
# ==============================================

echo ""
info "現在の Git LFS 設定:"
echo ""
git lfs env | head -20
echo ""

# ==============================================
# 完了
# ==============================================

echo ""
success "Git LFS セットアップが完了しました！"
echo ""
echo "次のステップ:"
echo "  1. .lfsconfig と .gitattributes をコミット"
echo "     git add .lfsconfig .gitattributes"
echo "     git commit -m 'Configure Git LFS with cargohold'"
echo ""
echo "  2. Git LFS の動作確認"
echo "     git lfs ls-files     # 追跡中のファイル一覧"
echo "     git lfs status       # 状態確認"
echo ""
echo "  3. ファイルの追加"
echo "     git add large-file.png"
echo "     git commit -m 'Add large file'"
echo "     git push"
echo ""
