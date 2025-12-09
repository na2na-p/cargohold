package domain

import (
	"encoding/json"
)

type ProviderType struct {
	value string
}

var (
	ProviderTypeGitHub = ProviderType{value: "github"}
)

var validProviderTypes = map[string]ProviderType{
	"github": ProviderTypeGitHub,
}

func NewProviderType(s string) (ProviderType, error) {
	pt, ok := validProviderTypes[s]
	if !ok {
		return ProviderType{}, ErrInvalidProviderType
	}
	return pt, nil
}

func (p ProviderType) String() string {
	return p.value
}

func (p ProviderType) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

func (p *ProviderType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	pt, ok := validProviderTypes[s]
	if !ok {
		return ErrInvalidProviderType
	}
	p.value = pt.value
	return nil
}
