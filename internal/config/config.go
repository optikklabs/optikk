// Package config loads CLI configuration, merging a file, env vars, and flags.
package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ClusterName and Namespace are fixed for the optikk stack across targets.
const (
	ClusterName = "optikk"
	Namespace   = "optikk"
)

// Target is the deployment environment a command acts on.
type Target string

const (
	TargetLocal Target = "local"
)

// Config is the merged CLI configuration. Flags override file/env values,
// which are applied by the root command after Load.
type Config struct {
	Target    Target `mapstructure:"target" yaml:"target"`
	DeployDir string `mapstructure:"deploy_dir" yaml:"deploy_dir"`
	Verbose   bool   `mapstructure:"verbose" yaml:"verbose"`

	Admin Admin `mapstructure:"admin" yaml:"admin"`

	// Data-CLI fields (Datadog Pup-style).
	ApiURL   string `mapstructure:"api_url" yaml:"api_url"` // OPTIKK_API_URL
	Output   string `mapstructure:"output" yaml:"output"`   // OPTIKK_OUTPUT (table|json|yaml)
	TenantID int64  `mapstructure:"team_id" yaml:"team_id"` // OPTIKK_TENANT_ID
	Token    string `mapstructure:"-" yaml:"-"`             // OPTIKK_TOKEN (runtime only)
}

// Admin holds the platform super-admin credentials seeded into query.
type Admin struct {
	Email    string `mapstructure:"email" yaml:"email"`
	Password string `mapstructure:"password" yaml:"password"`
}

// Load reads config from the given file (or the default search paths) plus
// OPTIKK_* env vars. A missing file is not an error — defaults are returned.
func Load(file string) (Config, error) {
	cfg := Config{Target: TargetLocal}
	path, err := resolveConfigFile(file)
	if err != nil {
		return Config{}, err
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	}
	applyEnv(&cfg)
	return cfg, nil
}

func resolveConfigFile(file string) (string, error) {
	if file != "" {
		if _, err := os.Stat(file); err != nil {
			return "", err
		}
		return file, nil
	}
	candidates := []string{"optikk.yaml"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".optikk", "config"))
		candidates = append(candidates, filepath.Join(home, ".optikk", "optikk.yaml"))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", nil
}

func applyEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("OPTIKK_TARGET")); v != "" {
		cfg.Target = Target(v)
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_DEPLOY_DIR")); v != "" {
		cfg.DeployDir = v
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_VERBOSE")); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.Verbose = parsed
		}
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_ADMIN_EMAIL")); v != "" {
		cfg.Admin.Email = v
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_ADMIN_PASSWORD")); v != "" {
		cfg.Admin.Password = v
	}

	// Data-CLI env vars (Datadog Pup-style).
	if v := strings.TrimSpace(os.Getenv("OPTIKK_API_URL")); v != "" {
		cfg.ApiURL = v
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_OUTPUT")); v != "" {
		cfg.Output = v
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_TENANT_ID")); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.TenantID = parsed
		}
	}
	if v := strings.TrimSpace(os.Getenv("OPTIKK_TOKEN")); v != "" {
		cfg.Token = v
	}
}
