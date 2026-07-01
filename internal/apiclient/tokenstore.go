package apiclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// cachedToken is the on-disk admin session at ~/.optikk/token.json.
type cachedToken struct {
	APIBase string `json:"apiBase"`
	Token   string `json:"token"`
}

func tokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".optikk", "token.json"), nil
}

// SaveToken persists the admin JWT for later team commands.
func SaveToken(apiBase, token string) error {
	path, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(cachedToken{APIBase: apiBase, Token: token})
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// LoadToken returns the cached JWT, erroring if none is present.
func LoadToken() (apiBase, token string, err error) {
	path, err := tokenPath()
	if err != nil {
		return "", "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("no cached admin session; run: optikk admin login")
	}
	var c cachedToken
	if err := json.Unmarshal(b, &c); err != nil {
		return "", "", err
	}
	return c.APIBase, c.Token, nil
}
