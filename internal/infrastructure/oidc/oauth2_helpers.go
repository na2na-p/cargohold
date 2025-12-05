package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateState はOAuth2のstateパラメータを生成します
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("stateパラメータの生成に失敗しました: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
