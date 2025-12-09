package infrastructure

import "errors"

// ErrNotFound はリソースが見つからない場合のエラー
var ErrNotFound = errors.New("not found")
