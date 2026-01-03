package middleware

import (
	"net/url"
	"sort"
	"strings"
)

var sensitiveParams = map[string]struct{}{
	"session_id": {},
	"code":       {},
	"state":      {},
}

const maskValue = "***"

func MaskSensitiveParams(uri string) string {
	if uri == "" {
		return ""
	}

	parsedURL, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	query := parsedURL.Query()
	if len(query) == 0 {
		return uri
	}

	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		if _, isSensitive := sensitiveParams[key]; isSensitive {
			parts = append(parts, url.QueryEscape(key)+"="+maskValue)
		} else {
			for _, v := range query[key] {
				parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(v))
			}
		}
	}

	parsedURL.RawQuery = strings.Join(parts, "&")
	return parsedURL.String()
}
