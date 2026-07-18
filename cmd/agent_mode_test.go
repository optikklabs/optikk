package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/optikklabs/optikk/internal/apiclient"
	"github.com/optikklabs/optikk/internal/clierr"
	"github.com/spf13/cobra"
)

func TestSignupNonInteractiveNamesMissingFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(&bytes.Buffer{}) // not a TTY

	_, err := signupInteractive(cmd, apiclient.New("https://api.example.com"), "https://api.example.com",
		signupInput{Email: "a@b.c"}, false)

	var e *clierr.Error
	if !errors.As(err, &e) || e.Kind != clierr.Usage {
		t.Fatalf("want a usage error, got %v", err)
	}
	for _, flag := range []string{"--password", "--name", "--org", "--accept-terms"} {
		if !strings.Contains(e.Msg, flag) {
			t.Errorf("missing-flags message %q lacks %s", e.Msg, flag)
		}
	}
	if strings.Contains(e.Msg, "--email") {
		t.Error("message names --email even though it was provided")
	}
}

func TestWriteResultEmitsJSONInAgentMode(t *testing.T) {
	app := &App{AgentMode: true}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	doc := signupDoc{Status: signupStatusCreated, APIKey: "tk_x",
		Tenant: &tenantInfo{ID: 7, Name: "acme"},
		OTLP:   newOTLPInfo("https://api.optikk.in", "tk_x")}
	humanCalled := false
	if err := writeResult(cmd, app, doc, func(w io.Writer) { humanCalled = true }); err != nil {
		t.Fatal(err)
	}
	if humanCalled {
		t.Error("human renderer ran in agent mode")
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("agent-mode output is not JSON: %v\n%s", err, out.String())
	}
	if got["status"] != "created" || got["api_key"] != "tk_x" {
		t.Errorf("doc = %v", got)
	}
	otlp := got["otlp"].(map[string]any)
	if otlp["endpoint"] != "https://ingest.optikk.in:4318" || otlp["headers"] != "x-api-key=tk_x" {
		t.Errorf("otlp = %v", otlp)
	}
}

func TestWriteResultUsesHumanRendererOnTable(t *testing.T) {
	app := &App{} // no agent mode; Cfg.Output empty
	app.Cfg.Output = "table"
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	if err := writeResult(cmd, app, struct{}{}, func(w io.Writer) { w.Write([]byte("human\n")) }); err != nil {
		t.Fatal(err)
	}
	if out.String() != "human\n" {
		t.Errorf("table output = %q, want the human renderer's text", out.String())
	}
}

func TestAgentSchemaIsParseable(t *testing.T) {
	root := NewRootCmd()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"agent", "schema"})
	if err := root.Execute(); err != nil {
		t.Fatalf("agent schema: %v", err)
	}

	var schema AgentSchema
	if err := json.Unmarshal(out.Bytes(), &schema); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	if schema.Version != "1.1" {
		t.Errorf("version = %q, want 1.1", schema.Version)
	}
	if len(schema.ExitCodes) == 0 || len(schema.Examples) == 0 {
		t.Error("schema missing exit_codes or examples")
	}
	var names []string
	for _, c := range schema.Commands {
		names = append(names, c.Use)
	}
	joined := strings.Join(names, "\n")
	for _, want := range []string{"verify", "errors", "onboard", "logs trace <trace-id>"} {
		if !strings.Contains(joined, want) {
			t.Errorf("schema commands missing %q", want)
		}
	}
}
