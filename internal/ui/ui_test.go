package ui

import (
	"strings"
	"testing"
	"time"
)

func TestFZFThemeArgsUseRoyalNoirPalette(t *testing.T) {
	args := strings.Join(FZFThemeArgs("ssh> "), "\n")

	expected := []string{
		"--prompt=ssh> ",
		"bg:#05020a",
		"hl:#d4af37",
		"marker:#7c3aed",
		"prompt:#c4b5fd",
		"border:#3b235c",
	}

	for _, value := range expected {
		if !strings.Contains(args, value) {
			t.Fatalf("FZFThemeArgs missing %q in %s", value, args)
		}
	}
}

func TestRenderRoyalLoaderFrame(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalLoaderFrame(LoaderOptions{
		Action:  "connecting",
		Subject: "api",
	}, 2, 1200*time.Millisecond)

	expected := []string{
		"[░░█████░░░░░░░░░░░]",
		"connecting",
		"api",
	}
	for _, value := range expected {
		if !strings.Contains(frame, value) {
			t.Fatalf("loader frame missing %q in %q", value, frame)
		}
	}
	unexpected := []string{"royal-noir", "ssh handoff", "1.2s"}
	for _, value := range unexpected {
		if strings.Contains(frame, value) {
			t.Fatalf("loader frame should not include %q in %q", value, frame)
		}
	}
}

func TestRenderRoyalLoaderResult(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalLoaderResult(LoaderOptions{
		Subject: "workspace-task_600_8048",
	}, true)

	expected := []string{
		"[██████████████████]",
		"done",
		"workspace-task_600_8048",
	}
	for _, value := range expected {
		if !strings.Contains(frame, value) {
			t.Fatalf("loader result missing %q in %q", value, frame)
		}
	}
}

func TestRenderRoyalLoaderResultUsesCustomSuccessAction(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalLoaderResult(LoaderOptions{
		Subject:       "adami",
		SuccessAction: "imported",
	}, true)

	if !strings.Contains(frame, "imported") || strings.Contains(frame, "done") {
		t.Fatalf("loader result should use custom action: %q", frame)
	}
}
