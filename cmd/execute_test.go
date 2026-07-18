package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/optikklabs/optikk/internal/clierr"
)

// runCLI executes the root command in-process and returns the resulting
// error's exit code plus captured stdout.
func runCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(&bytes.Buffer{}) // never a TTY in tests
	root.SetArgs(args)
	err := root.Execute()
	if err == nil {
		return 0, out.String()
	}
	return clierr.ExitCode(err), out.String()
}

// isolateHome keeps the developer's real ~/.optikk out of every CLI test.
func isolateHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestExitCodesEndToEnd(t *testing.T) {
	isolateHome(t)

	cases := []struct {
		name string
		args []string
		env  map[string]string
		want int
	}{
		{name: "onboard without flags is usage", args: []string{"--agent", "onboard"}, want: 2},
		{name: "signup without flags is usage", args: []string{"--agent", "signup"}, want: 2},
		{name: "auth login without flags is usage", args: []string{"--agent", "auth", "login"}, want: 2},
		{name: "delete without --yes is usage even in agent mode", args: []string{"--agent", "dashboards", "delete", "1"}, want: 2},
		{name: "monitors delete without --yes is usage", args: []string{"--agent", "monitors", "delete", "1"}, want: 2},
		{name: "keys revoke without --yes is usage", args: []string{"--agent", "keys", "revoke"}, want: 2},
		{name: "unknown flag is usage", args: []string{"--agent", "traces", "search", "--nope"}, want: 2},
		{name: "unauthenticated data command is auth", args: []string{"--agent", "traces", "search"}, want: 3},
		{name: "unreachable api is network", args: []string{"--agent", "--api-url", "https://127.0.0.1:1", "errors", "list"},
			env: map[string]string{"OPTIKK_TOKEN": "x"}, want: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			got, out := runCLI(t, tc.args...)
			if got != tc.want {
				t.Errorf("exit = %d, want %d (output: %s)", got, tc.want, out)
			}
		})
	}
}

func TestAgentSetupIsIdempotent(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()

	for i := 0; i < 2; i++ {
		if code, out := runCLI(t, "--agent", "agent", "setup", "--dir", dir, "--agents-md"); code != 0 {
			t.Fatalf("setup run %d exited %d: %s", i+1, code, out)
		}
	}

	skill, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "optikk", "SKILL.md"))
	if err != nil {
		t.Fatalf("skill file missing: %v", err)
	}
	if !strings.HasPrefix(string(skill), "---\nname: optikk\n") {
		t.Error("skill file missing frontmatter")
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md missing: %v", err)
	}
	if got := strings.Count(string(agents), "BEGIN optikk-cli"); got != 1 {
		t.Errorf("AGENTS.md has %d marked blocks after two runs, want 1", got)
	}
}

func TestAgentSetupPrintWritesNothing(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	code, out := runCLI(t, "agent", "setup", "--dir", dir, "--print")
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out, "# Operating Optikk with the optikk CLI") {
		t.Error("--print did not render the guide")
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Error("--print wrote files")
	}
}
