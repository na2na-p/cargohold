package domain

import (
	"errors"
	"fmt"
	"net/url"
)

var ErrEmptyAllowedRedirectURIs = errors.New("allowed redirect URIs cannot be empty")
var ErrInvalidRedirectURIFormat = errors.New("invalid redirect URI format")

type AllowedRedirectURIs struct {
	uris []string
}

func NewAllowedRedirectURIs(uris []string) (*AllowedRedirectURIs, error) {
	if len(uris) == 0 {
		return nil, ErrEmptyAllowedRedirectURIs
	}

	validatedURIs := make([]string, 0, len(uris))
	for _, uri := range uris {
		if uri == "" {
			return nil, fmt.Errorf("%w: empty URI", ErrInvalidRedirectURIFormat)
		}
		parsedURL, err := url.Parse(uri)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidRedirectURIFormat, uri)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return nil, fmt.Errorf("%w: missing scheme or host: %s", ErrInvalidRedirectURIFormat, uri)
		}
		validatedURIs = append(validatedURIs, uri)
	}

	return &AllowedRedirectURIs{
		uris: validatedURIs,
	}, nil
}

func (a *AllowedRedirectURIs) Contains(uri string) bool {
	for _, allowedURI := range a.uris {
		if uri == allowedURI {
			return true
		}
	}
	return false
}

func (a *AllowedRedirectURIs) Values() []string {
	result := make([]string, len(a.uris))
	copy(result, a.uris)
	return result
}
