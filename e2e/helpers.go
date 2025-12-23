//go:build e2e

// Package e2e はE2Eテストで使用するヘルパー関数を提供します
package e2e

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
)

var (
	// testPrivateKey はE2Eテスト用の固定秘密鍵（シングルトン）
	testPrivateKey     *rsa.PrivateKey
	testPrivateKeyOnce sync.Once
	testPrivateKeyErr  error

	// setupOnce はE2E環境セットアップを一度だけ実行するためのSync.Once
	setupOnce sync.Once
	setupErr  error

	// testRepositories はE2Eテスト用に許可リストに登録するリポジトリ一覧
	testRepositories = []string{
		"na2na-p/test-repo",
		"na2na-p/na2na-platform",
	}
)

// TestMain はE2Eテストパッケージ全体の初期化とクリーンアップを行います
func TestMain(m *testing.M) {
	if err := SetupE2EEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "E2Eテスト環境のセットアップに失敗しました: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := TeardownE2EEnvironment(); err != nil {
		fmt.Fprintf(os.Stderr, "E2Eテスト環境のクリーンアップに失敗しました: %v\n", err)
	}

	os.Exit(code)
}

// SetupE2EEnvironment はE2Eテスト環境をセットアップします
// この関数は各テストの前に呼び出されるべきです
// sync.Onceにより、複数回呼び出されても実際のセットアップは一度だけ実行されます
func SetupE2EEnvironment() error {
	setupOnce.Do(func() {
		setupErr = doSetupE2EEnvironment()
	})
	return setupErr
}

// doSetupE2EEnvironment は実際のセットアップ処理を行います
func doSetupE2EEnvironment() error {
	tempDir := filepath.Join(os.TempDir(), "cargohold-e2e-test")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("一時ディレクトリの作成に失敗しました: %w", err)
	}

	_ = os.Setenv("E2E_TEST_TEMP_DIR", tempDir)

	if err := registerTestRepositories(); err != nil {
		return fmt.Errorf("テスト用リポジトリの登録に失敗しました: %w", err)
	}

	return nil
}

// registerTestRepositories はテスト用リポジトリをDBに登録します
func registerTestRepositories() error {
	dbHost := getEnvOrDefault("DATABASE_HOST", "localhost")
	dbPort := getEnvOrDefault("DATABASE_PORT", "5432")
	dbUser := getEnvOrDefault("DATABASE_USER", "cargohold")
	dbPassword := getEnvOrDefault("DATABASE_PASSWORD", "cargohold_dev_password")
	dbName := getEnvOrDefault("DATABASE_DBNAME", "cargohold")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("データベース接続に失敗しました: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("データベースへの接続確認に失敗しました: %w", err)
	}

	for _, repo := range testRepositories {
		_, err := db.Exec(
			`INSERT INTO repository_allowlist (repository, created_at)
			 VALUES ($1, NOW())
			 ON CONFLICT (repository) DO NOTHING`,
			repo,
		)
		if err != nil {
			return fmt.Errorf("リポジトリ %s の登録に失敗しました: %w", repo, err)
		}
	}

	return nil
}

// getEnvOrDefault は環境変数を取得し、存在しない場合はデフォルト値を返します
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetBaseEndpoint はE2Eテスト対象のベースエンドポイントを返します
// 環境変数 E2E_TEST_ENDPOINT が設定されている場合はその値を使用し、
// 設定されていない場合は http://localhost:8080 をデフォルトとして返します
func GetBaseEndpoint() string {
	return getEnvOrDefault("E2E_TEST_ENDPOINT", "http://localhost:8080")
}

// GetBatchEndpoint はリポジトリのBatch APIエンドポイントを返します
func GetBatchEndpoint(repo string) string {
	return fmt.Sprintf("%s/%s/info/lfs/objects/batch", GetBaseEndpoint(), repo)
}

// GetVerifyEndpoint はリポジトリのVerify APIエンドポイントを返します
func GetVerifyEndpoint(repo string) string {
	return fmt.Sprintf("%s/%s/info/lfs/objects/verify", GetBaseEndpoint(), repo)
}

// TeardownE2EEnvironment はE2Eテスト環境をクリーンアップします
// この関数は各テストの後に呼び出されるべきです
func TeardownE2EEnvironment() error {
	tempDir := os.Getenv("E2E_TEST_TEMP_DIR")
	if tempDir != "" {
		if err := os.RemoveAll(tempDir); err != nil {
			return fmt.Errorf("一時ディレクトリの削除に失敗しました: %w", err)
		}
	}

	_ = os.Unsetenv("E2E_TEST_TEMP_DIR")

	return nil
}

// WaitForService は指定されたURLのサービスが利用可能になるまで待機します
// タイムアウトに達した場合はエラーを返します
func WaitForService(url string, timeout time.Duration) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	deadline := time.Now().Add(timeout)

	checkService := func() bool {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return true
			}
		}
		return false
	}

	if checkService() {
		return nil
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("サービスの起動を待機中にタイムアウトしました: %s", url)
		}

		select {
		case <-ticker.C:
			if checkService() {
				return nil
			}
		}
	}
}

// CreateTestFile は指定されたサイズのテスト用ファイルを生成します
// ファイルパスとエラーを返します
func CreateTestFile(size int64) (string, error) {
	tempDir := os.Getenv("E2E_TEST_TEMP_DIR")
	if tempDir == "" {
		return "", fmt.Errorf("E2E_TEST_TEMP_DIR環境変数が設定されていません")
	}

	filename := fmt.Sprintf("test-file-%d.bin", time.Now().UnixNano())
	filePath := filepath.Join(tempDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("テストファイルの作成に失敗しました: %w", err)
	}
	defer func() { _ = file.Close() }()

	written, err := io.CopyN(file, rand.Reader, size)
	if err != nil {
		return "", fmt.Errorf("テストファイルへのデータ書き込みに失敗しました: %w", err)
	}

	if written != size {
		return "", fmt.Errorf("書き込みサイズが不一致です: expected=%d, actual=%d", size, written)
	}

	return filePath, nil
}

// TestKeyPair はE2Eテスト用のRSA鍵ペアを保持します
type TestKeyPair struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// GenerateTestKeyPair はテスト用のRSA鍵ペアを生成します
func GenerateTestKeyPair() (*TestKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("RSA秘密鍵の生成に失敗しました: %w", err)
	}

	return &TestKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}, nil
}

// GenerateJWTWithKey はテスト用のJWTトークンをRS256で生成します
// claimsにはカスタムクレームを含めることができます
// 公開鍵も返すため、JWKSモックで使用できます
func GenerateJWTWithKey(claims map[string]interface{}) (string, *rsa.PublicKey, error) {
	keyPair, err := GenerateTestKeyPair()
	if err != nil {
		return "", nil, err
	}

	token, err := GenerateJWTWithKeyPair(claims, keyPair)
	if err != nil {
		return "", nil, err
	}

	return token, keyPair.PublicKey, nil
}

// GenerateJWTWithKeyPair は指定された鍵ペアを使用してJWTトークンを生成します
func GenerateJWTWithKeyPair(claims map[string]interface{}, keyPair *TestKeyPair) (string, error) {
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
		"nbf": now.Unix(),
	}

	for key, value := range claims {
		jwtClaims[key] = value
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	token.Header["kid"] = "test-key-id"

	signedToken, err := token.SignedString(keyPair.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("JWTトークンの署名に失敗しました: %w", err)
	}

	return signedToken, nil
}

// getTestPrivateKey は固定テスト用秘密鍵を返します（遅延読み込み、シングルトン）
func getTestPrivateKey() (*rsa.PrivateKey, error) {
	testPrivateKeyOnce.Do(func() {
		testPrivateKey, testPrivateKeyErr = loadTestPrivateKey()
	})
	return testPrivateKey, testPrivateKeyErr
}

// loadTestPrivateKey はe2e/testdata/test_private_key.pemから秘密鍵を読み込みます
func loadTestPrivateKey() (*rsa.PrivateKey, error) {
	paths := []string{
		"e2e/testdata/test_private_key.pem",
		"testdata/test_private_key.pem",
		"../e2e/testdata/test_private_key.pem",
	}

	var pemData []byte
	var err error
	for _, path := range paths {
		pemData, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if pemData == nil {
		return nil, fmt.Errorf("テスト用秘密鍵が見つかりません。go run ./cmd/generate-test-keys/ を実行してください")
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("PEMブロックのデコードに失敗しました")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("秘密鍵のパースに失敗しました: %w", err)
	}

	return privateKey, nil
}

// GenerateJWT はテスト用のJWTトークンをRS256で生成します
// 固定テスト用秘密鍵を使用し、JWKSモックと連携して検証可能なトークンを生成します
func GenerateJWT(claims map[string]interface{}) (string, error) {
	privateKey, err := getTestPrivateKey()
	if err != nil {
		return "", err
	}

	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(1 * time.Hour).Unix(),
		"nbf": now.Unix(),
	}

	for key, value := range claims {
		jwtClaims[key] = value
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	token.Header["kid"] = "test-key-id" // JWKSモックと一致させる

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("JWTトークンの署名に失敗しました: %w", err)
	}

	return signedToken, nil
}

// GetTestTempDir はE2Eテスト用の一時ディレクトリパスを返します
func GetTestTempDir() (string, error) {
	tempDir := os.Getenv("E2E_TEST_TEMP_DIR")
	if tempDir == "" {
		return "", fmt.Errorf("E2E_TEST_TEMP_DIR環境変数が設定されていません")
	}
	return tempDir, nil
}

// CleanupTestFiles は指定されたファイルパスのファイルを削除します
func CleanupTestFiles(paths ...string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("ファイルの削除に失敗しました: %s: %w", path, err)
		}
	}
	return nil
}

// CalculateFileHash はファイルのSHA-256ハッシュとサイズを計算します
// 戻り値: (ハッシュ文字列, ファイルサイズ, エラー)
func CalculateFileHash(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("ファイルを開けませんでした: %w", err)
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("ハッシュ計算中にエラーが発生しました: %w", err)
	}

	hashString := hex.EncodeToString(hash.Sum(nil))
	return hashString, size, nil
}

// GetOAuthLoginEndpoint はGitHub OAuth認証のログインエンドポイントURLを返します
func GetOAuthLoginEndpoint(repository string) string {
	return fmt.Sprintf("%s/auth/github/login?repository=%s", GetBaseEndpoint(), repository)
}
