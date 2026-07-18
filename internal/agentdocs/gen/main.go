// Command gen renders the repo-root AGENTS.md from the agentdocs template.
// Run it via `go generate ./...` (wired in agentdocs.go); a test asserts the
// checked-in file matches the template, so the two cannot drift.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/optikklabs/optikk/internal/agentdocs"
)

func main() {
	out := flag.String("o", "AGENTS.md", "output path")
	flag.Parse()
	guide, err := agentdocs.Guide(agentdocs.RepoDocVersion)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, []byte(guide), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gen:", err)
		os.Exit(1)
	}
}
