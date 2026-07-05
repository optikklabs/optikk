package cmd

import "testing"

// The cloud subtree must default to managed; the top-level tree must not.
func TestCloudSubtreeIsManaged(t *testing.T) {
	root := NewRootCmd()

	cloudSignup, _, err := root.Find([]string{"cloud", "signup"})
	if err != nil {
		t.Fatalf("find cloud signup: %v", err)
	}
	if !isManaged(cloudSignup) {
		t.Error("cloud signup should be managed")
	}

	localSignup, _, err := root.Find([]string{"signup"})
	if err != nil {
		t.Fatalf("find signup: %v", err)
	}
	if isManaged(localSignup) {
		t.Error("top-level signup should not be managed")
	}
}

// Provisioning commands are local-only; they must not appear under cloud.
func TestCloudHasNoOpsCommands(t *testing.T) {
	root := NewRootCmd()
	cloud, _, err := root.Find([]string{"cloud"})
	if err != nil {
		t.Fatalf("find cloud: %v", err)
	}
	// cobra.Find returns the parent itself when no subcommand matches.
	for _, ops := range []string{"up", "down", "status", "verify", "tenant", "admin"} {
		if c, _, _ := cloud.Find([]string{ops}); c != cloud {
			t.Errorf("cloud should not expose ops command %q", ops)
		}
	}
}
