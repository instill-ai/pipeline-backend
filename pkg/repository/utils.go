package repository

import (
	"encoding/base64"
	"encoding/json"
)

func transformBoolToDescString(b bool) string {
	if b {
		return " DESC"
	}
	return ""
}

// TODO: we should refactor this to have a flexible format and merge it into x package.

// DecodeToken decodes the token string into create_time and UUID
func DecodeToken(encodedToken string) (map[string]any, error) {
	byt, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, err
	}
	tokens := map[string]any{}
	err = json.Unmarshal(byt, &tokens)
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// EncodeToken encodes create_time and UUID into a single string
func EncodeToken(tokens map[string]any) (string, error) {
	b, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
