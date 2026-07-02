package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesYAMLAndEnv(t *testing.T) {
	t.Setenv("OPTIKK_TARGET", "local")
	t.Setenv("OPTIKK_DEPLOY_DIR", "/env/deploy")
	t.Setenv("OPTIKK_VERBOSE", "true")
	t.Setenv("OPTIKK_ADMIN_EMAIL", "env-admin@example.com")

	path := filepath.Join(t.TempDir(), "optikk.yaml")
	if err := os.WriteFile(path, []byte(`
target: local
deploy_dir: /yaml/deploy
admin:
  email: yaml-admin@example.com
  password: yaml-password
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Target != TargetLocal {
		t.Fatalf("target = %q, want %q", cfg.Target, TargetLocal)
	}
	if cfg.DeployDir != "/env/deploy" {
		t.Fatalf("deploy dir = %q, want env override", cfg.DeployDir)
	}
	if !cfg.Verbose {
		t.Fatal("verbose = false, want env override true")
	}
	if cfg.Admin.Email != "env-admin@example.com" {
		t.Fatalf("admin email = %q, want env override", cfg.Admin.Email)
	}
	if cfg.Admin.Password != "yaml-password" {
		t.Fatalf("admin password = %q, want YAML value", cfg.Admin.Password)
	}
}
