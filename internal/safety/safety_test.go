package safety

import "testing"

func TestValidateWorkspaceName(t *testing.T) {
	if err := ValidateWorkspaceName("feature-x"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
	if err := ValidateWorkspaceName("../feature"); err == nil {
		t.Fatal("expected path-like workspace name to be rejected")
	}
}

func TestCleanRelativePath(t *testing.T) {
	got, err := CleanRelativePath("src/../.env", "copy rule")
	if err == nil || got != "" {
		t.Fatalf("expected unsafe path rejection, got %q %v", got, err)
	}

	got, err = CleanRelativePath("./src/env.ts", "copy rule")
	if err != nil {
		t.Fatalf("valid relative path rejected: %v", err)
	}
	if got != "src/env.ts" {
		t.Fatalf("clean path = %q", got)
	}
}

func TestIsDirectChild(t *testing.T) {
	if !IsDirectChild("/workspaces", "/workspaces/workspace-a") {
		t.Fatal("expected direct child")
	}
	if IsDirectChild("/workspaces", "/workspaces/a/b") {
		t.Fatal("nested child should not be direct")
	}
	if IsDirectChild("/workspaces", "/workspace-other") {
		t.Fatal("sibling prefix should not be inside parent")
	}
}
