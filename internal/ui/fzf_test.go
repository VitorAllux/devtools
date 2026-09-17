package ui

import (
	"strings"
	"testing"
)

func TestFZFHubArgsBuildsBrandedHub(t *testing.T) {
	hub := FZFHub{
		Prompt:       "ssh> ",
		Title:        "dvv ssh",
		Subtitle:     "2 configured target(s)",
		BorderLabel:  "dvv ssh ",
		Preview:      "printf test",
		PreviewLabel: "selected target",
		Shortcuts: []FZFShortcut{
			{Label: "Enter", Description: "connect"},
			{Key: "A", Label: "Shift+A", Description: "add"},
			{Key: "R", Label: "Shift+R", Description: "remove"},
		},
		ExtraArgs: FZFHiddenRowArgs(),
	}

	args := strings.Join(hub.Args(), "\n")
	expected := []string{
		"--prompt=ssh> ",
		"--border-label=",
		"--header=",
		"--header-first",
		"--expect=A,R",
		"--preview=printf test",
		"--preview-window=right,44%,border-rounded,wrap",
		"--preview-label=",
		"--with-nth=2..",
	}

	for _, value := range expected {
		if !strings.Contains(args, value) {
			t.Fatalf("FZFHub Args missing %q in %s", value, args)
		}
	}
}

func TestFZFHubKeepsShortcutsOutOfHeaderByDefault(t *testing.T) {
	hub := FZFHub{
		Title: "dvv ssh",
		Shortcuts: []FZFShortcut{
			{Key: "A", Label: "Shift+A", Description: "add"},
		},
	}

	header := hub.Header()
	if strings.Contains(header, "Shift+A") {
		t.Fatalf("header should not include shortcut deck by default: %s", header)
	}

	hub.ShortcutsInHeader = true
	header = hub.Header()
	if !strings.Contains(header, "Shift+A") {
		t.Fatalf("header should include shortcut deck when enabled: %s", header)
	}
}

func TestFZFHubCanOverrideHeight(t *testing.T) {
	hub := FZFHub{
		Prompt:    "templates> ",
		Height:    "42%",
		MinHeight: "22",
	}

	args := strings.Join(hub.Args(), "\n")
	if strings.Contains(args, "--height=~85%") || strings.Contains(args, "--min-height=18") {
		t.Fatalf("FZFHub should replace default height args: %s", args)
	}
	for _, want := range []string{"--height=42%", "--min-height=22"} {
		if !strings.Contains(args, want) {
			t.Fatalf("FZFHub Args missing %q in %s", want, args)
		}
	}
}

func TestFZFHubAppliesConfiguredLayoutGlobally(t *testing.T) {
	defer SetFZFHubLayout(FZFHubLayout{})
	SetFZFHubLayout(FZFHubLayout{HeightPercent: 85, MinHeight: 24, PreviewWidthPercent: 42})

	hub := FZFHub{
		Height:        "42%",
		MinHeight:     "16",
		Preview:       "printf test",
		PreviewWindow: "right,34%,border-rounded,wrap",
	}
	args := strings.Join(hub.Args(), "\n")
	for _, want := range []string{"--height=85%", "--min-height=24", "--preview-window=right,42%,border-rounded,wrap"} {
		if !strings.Contains(args, want) {
			t.Fatalf("global FZF layout missing %q in %s", want, args)
		}
	}
}

func TestFZFHubIgnoresUnsafeConfiguredLayout(t *testing.T) {
	defer SetFZFHubLayout(FZFHubLayout{})
	SetFZFHubLayout(FZFHubLayout{HeightPercent: 10, MinHeight: 500, PreviewWidthPercent: 90})

	hub := FZFHub{
		Height:        "42%",
		MinHeight:     "16",
		Preview:       "printf test",
		PreviewWindow: "right,34%,border-rounded,wrap",
	}
	args := strings.Join(hub.Args(), "\n")
	for _, want := range []string{"--height=42%", "--min-height=16", "--preview-window=right,34%,border-rounded,wrap"} {
		if !strings.Contains(args, want) {
			t.Fatalf("fallback FZF layout missing %q in %s", want, args)
		}
	}
}

func TestFZFThemeArgsAppliesConfiguredHeightToNonHubSelectors(t *testing.T) {
	defer SetFZFHubLayout(FZFHubLayout{})
	SetFZFHubLayout(FZFHubLayout{HeightPercent: 85, MinHeight: 24})

	args := strings.Join(FZFThemeArgs("pick> "), "\n")
	for _, want := range []string{"--height=85%", "--min-height=24"} {
		if !strings.Contains(args, want) {
			t.Fatalf("configured selector layout missing %q in %s", want, args)
		}
	}
	if strings.Contains(args, "--height=~85%") {
		t.Fatalf("configured selector should not retain adaptive height: %s", args)
	}
}

func TestFZFPreviewCommandDeckPrintsEveryShortcut(t *testing.T) {
	deck := FZFPreviewCommandDeck([]FZFShortcut{
		{Label: "Enter", Description: "open"},
		{Key: "N", Label: "Shift+N", Description: "create workspace"},
		{Key: "D", Label: "Shift+D", Description: "delete selected"},
	})

	for _, want := range []string{"Enter", "open", "Shift+N", "create workspace", "Shift+D", "delete selected"} {
		if !strings.Contains(deck, want) {
			t.Fatalf("command deck missing %q: %s", want, deck)
		}
	}
	if strings.Contains(deck, "DVV_FZF_COMMANDS") {
		t.Fatalf("command deck should be static printf calls: %s", deck)
	}
}

func TestFZFSelectedRaw(t *testing.T) {
	raw := FZFSelectedRaw("api root@example.com\tstyled display")
	if raw != "api root@example.com" {
		t.Fatalf("raw = %q, want api root@example.com", raw)
	}

	raw = FZFSelectedRaw("api root@example.com")
	if raw != "api root@example.com" {
		t.Fatalf("raw = %q, want api root@example.com", raw)
	}
}

func TestParseFZFExpectOutputSupportsMultiSelect(t *testing.T) {
	key, selected := ParseFZFExpectOutput("D\n/path/a\trow a\n/path/b\trow b\n")
	if key != "D" {
		t.Fatalf("key = %q", key)
	}
	if len(selected) != 2 || selected[0] != "/path/a\trow a" || selected[1] != "/path/b\trow b" {
		t.Fatalf("selected = %#v", selected)
	}

	key, selected = ParseFZFExpectOutput("\n/path/a\trow a\n")
	if key != "" || len(selected) != 1 || selected[0] != "/path/a\trow a" {
		t.Fatalf("got key=%q selected=%#v", key, selected)
	}
}
