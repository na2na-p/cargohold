// Package main はE2Eテスト用のRSA鍵ペアとJWKS JSONを生成するツールです
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

func main() {
	outputDir := "e2e/testdata"
	privateKeyPath := filepath.Join(outputDir, "test_private_key.pem")
	jwksPath := filepath.Join(outputDir, "jwks.json")

	if fileExists(privateKeyPath) && fileExists(jwksPath) {
		fmt.Println("テスト鍵は既に存在します。スキップします。")
		return
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "ディレクトリの作成に失敗しました: %v\n", err)
		os.Exit(1)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RSA鍵ペアの生成に失敗しました: %v\n", err)
		os.Exit(1)
	}

	if err := savePrivateKey(privateKey, privateKeyPath); err != nil {
		fmt.Fprintf(os.Stderr, "秘密鍵の保存に失敗しました: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("秘密鍵を保存しました: %s\n", privateKeyPath)

	if err := saveJWKS(&privateKey.PublicKey, jwksPath); err != nil {
		fmt.Fprintf(os.Stderr, "JWKSの保存に失敗しました: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("JWKSを保存しました: %s\n", jwksPath)

	fmt.Println("\n鍵の生成が完了しました。")
	fmt.Println("これらのファイルはE2Eテスト専用です。本番環境では使用しないでください。")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func savePrivateKey(privateKey *rsa.PrivateKey, path string) error {
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// 秘密鍵ファイルは所有者のみ読み書き可能にする
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	return pem.Encode(file, privateKeyPEM)
}

func saveJWKS(publicKey *rsa.PublicKey, path string) error {
	n := base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes())
	e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(publicKey.E)).Bytes())

	jwks := JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Kid: "test-key-id",
				Alg: "RS256",
				N:   n,
				E:   e,
			},
		},
	}

	jsonData, err := json.MarshalIndent(jwks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, jsonData, 0644)
}
