package agentdocs

import (
	"os"
	"strings"
	"testing"
)

func TestSkillHasFrontmatter(t *testing.T) {
	skill, err := Skill("test")
	if err != nil {
		t.Fatalf("Skill: %v", err)
	}
	if !strings.HasPrefix(skill, "---\nname: optikk\n") {
		t.Errorf("skill missing frontmatter, starts: %q", skill[:40])
	}
	if !strings.Contains(skill, "optikk verify") || !strings.Contains(skill, "--accept-terms") {
		t.Error("skill guide is missing core content")
	}
}

func TestGuideRendersEveryPlaybook(t *testing.T) {
	guide, err := Guide("test")
	if err != nil {
		t.Fatalf("Guide: %v", err)
	}
	for _, ex := range Examples() {
		if !strings.Contains(guide, ex.Goal) {
			t.Errorf("guide missing playbook %q", ex.Goal)
		}
	}
	for code, meaning := range ExitCodes() {
		if !strings.Contains(guide, meaning) {
			t.Errorf("guide missing exit code %s (%s)", code, meaning)
		}
	}
}

func TestUpsertAgentsSectionIsIdempotent(t *testing.T) {
	once, err := UpsertAgentsSection("", "test")
	if err != nil {
		t.Fatalf("UpsertAgentsSection: %v", err)
	}
	twice, err := UpsertAgentsSection(once, "test")
	if err != nil {
		t.Fatalf("UpsertAgentsSection(second): %v", err)
	}
	if strings.Count(twice, BeginMarker) != 1 || strings.Count(twice, EndMarker) != 1 {
		t.Error("re-running setup duplicated the marked block")
	}
}

func TestUpsertAgentsSectionPreservesSurroundingContent(t *testing.T) {
	existing := "# My project\n\nHouse rules.\n"
	updated, err := UpsertAgentsSection(existing, "test")
	if err != nil {
		t.Fatalf("UpsertAgentsSection: %v", err)
	}
	if !strings.HasPrefix(updated, "# My project") || !strings.Contains(updated, "House rules.") {
		t.Error("existing content was lost")
	}
	again, err := UpsertAgentsSection(updated, "test")
	if err != nil {
		t.Fatalf("UpsertAgentsSection(again): %v", err)
	}
	if !strings.Contains(again, "House rules.") || strings.Count(again, BeginMarker) != 1 {
		t.Error("second upsert damaged the document")
	}
}

// TestRepoAgentsMDIsCurrent pins the checked-in AGENTS.md to the template.
// If this fails, run: make gen
func TestRepoAgentsMDIsCurrent(t *testing.T) {
	want, err := Guide(RepoDocVersion)
	if err != nil {
		t.Fatalf("Guide: %v", err)
	}
	got, err := os.ReadFile("../../AGENTS.md")
	if err != nil {
		t.Fatalf("read AGENTS.md: %v (run: make gen)", err)
	}
	if string(got) != want {
		t.Error("AGENTS.md is stale — run: make gen")
	}
}
