package cmd

import "testing"

func TestOTLPEndpoint(t *testing.T) {
	cases := []struct {
		name    string
		apiBase string
		want    string
	}{
		{name: "localhost", apiBase: "http://localhost:8080", want: "http://localhost:4318"},
		{name: "hosted swaps api to ingest", apiBase: "https://api.optikk.in", want: "https://ingest.optikk.in:4318"},
		{name: "bare ip keeps host", apiBase: "http://10.0.0.5", want: "http://10.0.0.5:4318"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := otlpEndpoint(tc.apiBase); got != tc.want {
				t.Errorf("otlpEndpoint(%q) = %q, want %q", tc.apiBase, got, tc.want)
			}
		})
	}
}

func TestOTLPEndpointEnvOverride(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "https://custom:9999")
	if got := otlpEndpoint("https://api.optikk.in"); got != "https://custom:9999" {
		t.Errorf("otlpEndpoint with env override = %q, want the env value", got)
	}
}
