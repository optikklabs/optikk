// Package verify runs the health + trace-roundtrip check against a stack.
package verify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/optikklabs/optikk/internal/kubectl"
)

const spanCountQuery = "SELECT count() FROM optikk.spans"

// Options configures a verification run.
type Options struct {
	Kube      kubectl.Kube
	Namespace string
	APIBase   string
	OTLPBase  string
	APIKey    string
	TraceFile string
	Out       io.Writer
}

// Run checks /health, sends one OTLP trace, and confirms the span count rose.
func Run(ctx context.Context, o Options) error {
	log(o.Out, "checking %s/health", o.APIBase)
	if err := checkHealth(ctx, o.APIBase); err != nil {
		return err
	}

	before, err := execCHCount(ctx, o.Kube, o.Namespace, spanCountQuery)
	if err != nil {
		return fmt.Errorf("count spans (before): %w", err)
	}
	log(o.Out, "span count before: %d", before)

	log(o.Out, "posting trace %s -> %s/v1/traces", o.TraceFile, o.OTLPBase)
	if err := postTrace(ctx, o.OTLPBase, o.APIKey, o.TraceFile); err != nil {
		return err
	}

	after, err := waitForIncrease(ctx, o, before)
	if err != nil {
		return err
	}
	log(o.Out, "span count after: %d (+%d) — roundtrip OK", after, after-before)
	return nil
}

func checkHealth(ctx context.Context, apiBase string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiBase+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("health request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned %s", resp.Status)
	}
	return nil
}

func postTrace(ctx context.Context, otlpBase, apiKey, traceFile string) error {
	body, err := os.ReadFile(traceFile)
	if err != nil {
		return fmt.Errorf("read trace file: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, otlpBase+"/v1/traces", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("post trace: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post trace returned %s: %s", resp.Status, string(msg))
	}
	return nil
}

// waitForIncrease polls the span count until it exceeds before (ingestion is
// async: OTLP -> collector -> ingest -> mq -> ClickHouse).
func waitForIncrease(ctx context.Context, o Options, before int64) (int64, error) {
	deadline := time.Now().Add(30 * time.Second)
	for {
		after, err := execCHCount(ctx, o.Kube, o.Namespace, spanCountQuery)
		if err != nil {
			return 0, err
		}
		if after > before {
			return after, nil
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("span count did not increase within 30s (still %d)", after)
		}
		time.Sleep(2 * time.Second)
	}
}

func log(w io.Writer, format string, args ...any) {
	if w != nil {
		fmt.Fprintf(w, format+"\n", args...)
	}
}
