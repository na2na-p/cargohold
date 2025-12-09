package domain

import (
	"time"
)

type Token struct {
	accessToken  string
	refreshToken string
	idToken      string
	expiresAt    time.Time
}

func NewToken(accessToken, refreshToken, idToken string, expiresAt time.Time) *Token {
	return &Token{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		idToken:      idToken,
		expiresAt:    expiresAt,
	}
}

func (t *Token) AccessToken() string {
	return t.accessToken
}

func (t *Token) RefreshToken() string {
	return t.refreshToken
}

func (t *Token) IDToken() string {
	return t.idToken
}

func (t *Token) ExpiresAt() time.Time {
	return t.expiresAt
}
