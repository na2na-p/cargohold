package s3

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// GenerateStorageKey は、オブジェクトIDとハッシュアルゴリズムからS3オブジェクトキーを生成します。
// 形式: objects/{hash_algo}/{oid[0:2]}/{oid[2:4]}/{oid}
// 例: objects/sha256/ab/cd/abcdef123456...
var (
	ErrInvalidOID      = errors.New("invalid oid")
	ErrInvalidHashAlgo = errors.New("invalid hash algorithm")
)

var (
	hashAlgoPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	oidPattern      = regexp.MustCompile(`^[a-fA-F0-9]+$`)
)

func validateHashAlgo(hashAlgo string) error {
	if hashAlgo == "" {
		return ErrInvalidHashAlgo
	}
	if strings.Contains(hashAlgo, "..") {
		return ErrInvalidHashAlgo
	}
	if !hashAlgoPattern.MatchString(hashAlgo) {
		return ErrInvalidHashAlgo
	}
	return nil
}

func validateOID(oid string) error {
	if oid == "" {
		return ErrInvalidOID
	}
	if len(oid) < 4 {
		return ErrInvalidOID
	}
	if !oidPattern.MatchString(oid) {
		return ErrInvalidOID
	}
	return nil
}

func GenerateStorageKey(oid, hashAlgo string) (string, error) {
	if err := validateHashAlgo(hashAlgo); err != nil {
		return "", err
	}

	if err := validateOID(oid); err != nil {
		return "", err
	}

	return fmt.Sprintf("objects/%s/%s/%s/%s",
		hashAlgo,
		oid[0:2],
		oid[2:4],
		oid,
	), nil
}
