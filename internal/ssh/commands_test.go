package ssh

import "testing"

func TestParseAddArgsSupportsFlagsAndPositionals(t *testing.T) {
	name, target, err := parseAddArgs([]string{"--name", "prod", "--conn", "deploy@example.com"})
	if err != nil {
		t.Fatalf("parseAddArgs flags returned error: %v", err)
	}
	if name != "prod" || target != "deploy@example.com" {
		t.Fatalf("flags got name=%q target=%q", name, target)
	}

	name, target, err = parseAddArgs([]string{"prod", "deploy@example.com"})
	if err != nil {
		t.Fatalf("parseAddArgs positionals returned error: %v", err)
	}
	if name != "prod" || target != "deploy@example.com" {
		t.Fatalf("positionals got name=%q target=%q", name, target)
	}
}

func TestParseRemoveArgsSupportsNamePositionals(t *testing.T) {
	name, line, err := parseRemoveArgs([]string{"prod"})
	if err != nil {
		t.Fatalf("parseRemoveArgs returned error: %v", err)
	}
	if name != "prod" || line != "" {
		t.Fatalf("got name=%q line=%q", name, line)
	}
}
