package ssh

import (
	"strings"
	"testing"
)

func TestParseFZFExpectOutput(t *testing.T) {
	key, selection := parseFZFExpectOutput("R\napi root@example.com\n")
	if key != "R" || selection != "api root@example.com" {
		t.Fatalf("got key=%q selection=%q", key, selection)
	}

	key, selection = parseFZFExpectOutput("\napi root@example.com\n")
	if key != "" || selection != "api root@example.com" {
		t.Fatalf("got key=%q selection=%q", key, selection)
	}
}

func TestStyledEntriesInputKeepsRawSelectionHidden(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	input := styledEntriesInput([]Entry{
		{Name: "api", Target: "root@example.com", Raw: "api root@example.com"},
	})

	if !strings.Contains(input, "__dvv_header__\t NO  TARGET") {
		t.Fatalf("styled input should keep the header aligned with the fzf pointer gutter: %q", input)
	}
	if !strings.Contains(input, "api root@example.com\t01") {
		t.Fatalf("styled input should keep raw value before tab: %q", input)
	}
	if !strings.Contains(input, "api") || !strings.Contains(input, "root") || !strings.Contains(input, "example.com") {
		t.Fatalf("styled input missing display data: %q", input)
	}
}

func TestSSHHubHeaderLinesShowsError(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	lines := sshHubHeaderLines("cannot reach example.com:22")
	if len(lines) != 1 {
		t.Fatalf("expected one error header line, got %#v", lines)
	}
	if !strings.Contains(lines[0], "error cannot reach example.com:22") {
		t.Fatalf("unexpected error header line: %q", lines[0])
	}
}

func TestSplitSSHTarget(t *testing.T) {
	user, host := splitSSHTarget("forge@10.120.0.208")
	if user != "forge" || host != "10.120.0.208" {
		t.Fatalf("got user=%q host=%q", user, host)
	}

	user, host = splitSSHTarget("example.com")
	if user != "-" || host != "example.com" {
		t.Fatalf("got user=%q host=%q", user, host)
	}
}
