package apiclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Context is one named CLI profile: which API, tenant, and session it targets.
type Context struct {
	APIURL   string `json:"api_url"`
	TenantID int64  `json:"team_id"`
	Token    string `json:"token"`
}

// cliConfig is the on-disk kubectl-style config at ~/.optikk/config.json.
type cliConfig struct {
	CurrentContext string             `json:"current_context"`
	Contexts       map[string]Context `json:"contexts"`
}

// legacyToken is the pre-contexts single session at ~/.optikk/token.json.
type legacyToken struct {
	APIBase string `json:"apiBase"`
	Token   string `json:"token"`
}

const defaultContextName = "default"

func optikkDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".optikk"), nil
}

func configPath() (string, error) {
	dir, err := optikkDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func legacyTokenPath() (string, error) {
	dir, err := optikkDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// loadConfig reads config.json, migrating a legacy token.json into a default
// context when config.json is absent. Returns an empty config if neither exists.
func loadConfig() (cliConfig, error) {
	path, err := configPath()
	if err != nil {
		return cliConfig{}, err
	}
	if b, err := os.ReadFile(path); err == nil {
		var cfg cliConfig
		if err := json.Unmarshal(b, &cfg); err != nil {
			return cliConfig{}, err
		}
		if cfg.Contexts == nil {
			cfg.Contexts = map[string]Context{}
		}
		return cfg, nil
	}
	return migrateLegacyToken()
}

// migrateLegacyToken folds an old token.json into an in-memory default context.
func migrateLegacyToken() (cliConfig, error) {
	cfg := cliConfig{CurrentContext: defaultContextName, Contexts: map[string]Context{}}
	path, err := legacyTokenPath()
	if err != nil {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, nil
	}
	var t legacyToken
	if err := json.Unmarshal(b, &t); err != nil {
		return cfg, nil
	}
	cfg.Contexts[defaultContextName] = Context{APIURL: t.APIBase, Token: t.Token}
	return cfg, nil
}

func saveConfig(cfg cliConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

func currentName(cfg cliConfig) string {
	if cfg.CurrentContext != "" {
		return cfg.CurrentContext
	}
	return defaultContextName
}

// SaveToken stores a session under the current context, preserving its tenant.
func SaveToken(apiBase, token string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	name := currentName(cfg)
	entry := cfg.Contexts[name]
	entry.APIURL = apiBase
	entry.Token = token
	cfg.Contexts[name] = entry
	cfg.CurrentContext = name
	return saveConfig(cfg)
}

// LoadToken returns the current context's API base and JWT.
func LoadToken() (apiBase, token string, err error) {
	cfg, err := loadConfig()
	if err != nil {
		return "", "", err
	}
	entry := cfg.Contexts[currentName(cfg)]
	if entry.Token == "" {
		return "", "", fmt.Errorf("no cached session; run: optikk login")
	}
	return entry.APIURL, entry.Token, nil
}

// CurrentContext returns the active context (tenant id included), for callers
// that need the default tenant beyond just the token.
func CurrentContext() (Context, error) {
	cfg, err := loadConfig()
	if err != nil {
		return Context{}, err
	}
	return cfg.Contexts[currentName(cfg)], nil
}

// SetContext upserts a context's API URL and tenant, preserving any token.
func SetContext(name, apiURL string, tenantID int64) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	entry := cfg.Contexts[name]
	entry.APIURL = apiURL
	entry.TenantID = tenantID
	cfg.Contexts[name] = entry
	if cfg.CurrentContext == "" {
		cfg.CurrentContext = name
	}
	return saveConfig(cfg)
}

// UseContext switches the active context, erroring if it does not exist.
func UseContext(name string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	if _, ok := cfg.Contexts[name]; !ok {
		return fmt.Errorf("no such context %q; create it with: optikk config set-context %s --api-url <url>", name, name)
	}
	cfg.CurrentContext = name
	return saveConfig(cfg)
}

// ListContexts returns all contexts (sorted names) and the current one.
func ListContexts() (names []string, contexts map[string]Context, current string, err error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, "", err
	}
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, cfg.Contexts, currentName(cfg), nil
}
