package apiclient

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateHome points ~/.optikk at a temp dir so tests never touch real config.
func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestContextRoundTrip(t *testing.T) {
	isolateHome(t)

	if err := SetContext("prod", "https://api.optikk.in", 4); err != nil {
		t.Fatalf("SetContext: %v", err)
	}
	if err := UseContext("prod"); err != nil {
		t.Fatalf("UseContext: %v", err)
	}
	if err := SaveToken("https://api.optikk.in", "jwt-123"); err != nil {
		t.Fatalf("SaveToken: %v", err)
	}

	ctx, err := CurrentContext()
	if err != nil {
		t.Fatalf("CurrentContext: %v", err)
	}
	if ctx.TenantID != 4 || ctx.Token != "jwt-123" || ctx.APIURL != "https://api.optikk.in" {
		t.Errorf("unexpected current context: %+v", ctx)
	}

	if err := UseContext("missing"); err == nil {
		t.Error("UseContext on unknown name should error")
	}
}

func TestLegacyTokenMigration(t *testing.T) {
	isolateHome(t)

	dir, _ := optikkDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	legacy := `{"apiBase":"http://localhost:8080","token":"old-jwt"}`
	if err := os.WriteFile(filepath.Join(dir, "token.json"), []byte(legacy), 0o600); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	base, token, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken after migration: %v", err)
	}
	if base != "http://localhost:8080" || token != "old-jwt" {
		t.Errorf("legacy not migrated: base=%q token=%q", base, token)
	}
}

func TestLoadTokenNoSession(t *testing.T) {
	isolateHome(t)
	if _, _, err := LoadToken(); err == nil {
		t.Error("LoadToken with no config should error")
	}
}
