package ports

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestParseSS(t *testing.T) {
	output := `LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("node",pid=1234,fd=22))
LISTEN 0 4096 [::1]:8000 [::]:* users:(("php",pid=4321,fd=8))`

	got := parseSS(output)
	want := []Entry{
		{Port: 3000, PID: 1234, Process: "node", Address: "127.0.0.1:3000"},
		{Port: 8000, PID: 4321, Process: "php", Address: "[::1]:8000"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseSS = %#v, want %#v", got, want)
	}
}

func TestParseLSOF(t *testing.T) {
	output := "p1234\ncnode\nn*:3000\np4321\ncphp\nnTCP 127.0.0.1:8000 (LISTEN)\n"

	got := parseLSOF(output)
	want := []Entry{
		{Port: 3000, PID: 1234, Process: "node", Address: "*:3000"},
		{Port: 8000, PID: 4321, Process: "php", Address: "TCP 127.0.0.1:8000 (LISTEN)"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseLSOF = %#v, want %#v", got, want)
	}
}

func TestPortRowsRenderUsefulColumns(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rows := portRows([]Entry{{Port: 3000, PID: 1234, Process: "ng serve", Address: "127.0.0.1:3000", Project: "web", URL: "http://localhost:3000", CWD: "/repo/web", Command: "ng serve --host localhost"}})
	for _, want := range []string{"PORT", "BIND", "PID", "PROCESS", "PROJECT", "3000", "127.0.0.1:3000", "ng serve", "web", "http://localhost:3000", "/repo/web", "ng serve --host localhost"} {
		if !strings.Contains(rows, want) {
			t.Fatalf("port rows missing %q: %q", want, rows)
		}
	}
}

func TestSelectedPortEntryUsesRawColumn(t *testing.T) {
	entries := []Entry{
		{Port: 3000, PID: 1234, Process: "node", Address: "127.0.0.1:3000"},
		{Port: 4200, PID: 5678, Process: "ng serve", Address: "127.0.0.1:4200"},
	}
	selected := []string{portLine(portRaw(entries[1]), "4200", "5678", "ng serve", "web", "http://localhost:4200", "127.0.0.1:4200", "/repo/web", "ng serve --host localhost", portRow(1, entries[1]))}

	got, ok := selectedPortEntry(entries, selected)
	if !ok || !reflect.DeepEqual(got, entries[1]) {
		t.Fatalf("selectedPortEntry = %#v, %v; want %#v, true", got, ok, entries[1])
	}
}

func TestListUsesSSAndAddsURL(t *testing.T) {
	runner := &portsFakeRunner{
		paths:   map[string]bool{"ss": true},
		outputs: []string{`LISTEN 0 4096 127.0.0.1:3000 0.0.0.0:* users:(("node",pid=1234,fd=22))`},
	}
	manager := &Manager{Config: &config.Config{Project: config.DefaultProjectConfig()}, Runner: runner}

	got, err := manager.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 || got[0].URL != "http://localhost:3000" || got[0].Project != "system" {
		t.Fatalf("entries = %#v", got)
	}
}

type portsFakeRunner struct {
	paths   map[string]bool
	outputs []string
	calls   []portsCall
}

type portsCall struct {
	dir  string
	name string
	args []string
}

func (r *portsFakeRunner) Run(_ context.Context, dir string, name string, args ...string) error {
	r.calls = append(r.calls, portsCall{dir: dir, name: name, args: append([]string{}, args...)})
	return nil
}

func (r *portsFakeRunner) Output(_ context.Context, dir string, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, portsCall{dir: dir, name: name, args: append([]string{}, args...)})
	if len(r.outputs) == 0 {
		return nil, nil
	}
	out := r.outputs[0]
	r.outputs = r.outputs[1:]
	return []byte(out), nil
}

func (r *portsFakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, nil
}

func (r *portsFakeRunner) Start(context.Context, string, string, ...string) error {
	return nil
}

func (r *portsFakeRunner) LookPath(name string) (string, error) {
	if r.paths != nil && r.paths[name] {
		return name, nil
	}
	return "", errNotFound{}
}

type errNotFound struct{}

func (errNotFound) Error() string { return "not found" }
