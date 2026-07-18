package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMergesYAMLAndEnv(t *testing.T) {
	t.Setenv("OPTIKK_API_URL", "http://env:18040")
	t.Setenv("OPTIKK_VERBOSE", "true")

	path := filepath.Join(t.TempDir(), "optikk.yaml")
	if err := os.WriteFile(path, []byte(`
api_url: http://yaml:18040
team_id: 42
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ApiURL != "http://env:18040" {
		t.Fatalf("api_url = %q, want env override", cfg.ApiURL)
	}
	if !cfg.Verbose {
		t.Fatal("verbose = false, want env override true")
	}
	if cfg.TenantID != 42 {
		t.Fatalf("tenant_id = %d, want YAML value", cfg.TenantID)
	}
}

func TestOptikkAgentEnv(t *testing.T) {
	// Point Load at an empty file so the developer's real ~/.optikk config
	// cannot leak into the test.
	path := filepath.Join(t.TempDir(), "optikk.yaml")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPTIKK_AGENT", "1")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AgentMode {
		t.Fatal("AgentMode = false, want true from OPTIKK_AGENT=1")
	}

	t.Setenv("OPTIKK_AGENT", "not-a-bool")
	cfg, err = Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AgentMode {
		t.Fatal("AgentMode = true, want false for an unparseable OPTIKK_AGENT")
	}
}
