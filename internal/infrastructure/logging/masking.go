package logging

import (
	"log/slog"
	"strings"
)

var defaultSensitiveKeys = []string{
	"token",
	"id_token",
	"idtoken",
	"access_token",
	"refresh_token",
	"authorization",
	"session_id",
	"sessionid",
	"oauth_state",
	"oidc_state",
	"password",
	"secret",
	"client_secret",
	"clientsecret",
	"secret_access_key",
	"secretaccesskey",
	"access_key_id",
	"accesskeyid",
	"api_key",
	"apikey",
	"credential",
	"private_key",
	"email",
	"oidc_subject",
}

type SensitiveMasker struct {
	sensitiveKeys map[string]bool
}

func NewSensitiveMasker(keys []string) *SensitiveMasker {
	m := make(map[string]bool, len(keys))
	for _, key := range keys {
		m[key] = true
	}
	return &SensitiveMasker{sensitiveKeys: m}
}

func (sm *SensitiveMasker) MaskAttrs(_ []string, a slog.Attr) slog.Attr {
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		maskedAttrs := make([]any, 0, len(attrs))
		for _, attr := range attrs {
			maskedAttrs = append(maskedAttrs, sm.MaskAttrs(nil, attr))
		}
		return slog.Group(a.Key, maskedAttrs...)
	}

	key := strings.ToLower(a.Key)

	if sm.sensitiveKeys[key] {
		return slog.String(a.Key, "[REDACTED]")
	}

	for sensitiveKey := range sm.sensitiveKeys {
		if strings.Contains(key, sensitiveKey) {
			return slog.String(a.Key, "[REDACTED]")
		}
	}

	return a
}

var defaultMasker = NewSensitiveMasker(defaultSensitiveKeys)

func MaskSensitiveAttrs(groups []string, a slog.Attr) slog.Attr {
	return defaultMasker.MaskAttrs(groups, a)
}
