package ssh

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestParseTargetEndpoint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		host string
		port string
		ok   bool
	}{
		{name: "user host", in: "forge@example.com", host: "example.com", port: "22", ok: true},
		{name: "plain host", in: "example.com", host: "example.com", port: "22", ok: true},
		{name: "host port", in: "forge@example.com:2222", host: "example.com", port: "2222", ok: true},
		{name: "bracket host port", in: "forge@[2001:db8::1]:2222", host: "2001:db8::1", port: "2222", ok: true},
		{name: "empty", in: "", ok: false},
		{name: "bad port", in: "example.com:99999", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			endpoint, ok := parseTargetEndpoint(tt.in)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if endpoint.Host != tt.host || endpoint.Port != tt.port {
				t.Fatalf("endpoint = %#v, want host=%q port=%q", endpoint, tt.host, tt.port)
			}
		})
	}
}

func TestParseSSHConfigEndpoint(t *testing.T) {
	output := "user forge\nhostname 10.120.0.208\nport 2222\n"
	endpoint, ok := parseSSHConfigEndpoint(output)
	if !ok {
		t.Fatal("expected SSH config endpoint")
	}
	if endpoint.Host != "10.120.0.208" || endpoint.Port != "2222" {
		t.Fatalf("endpoint = %#v", endpoint)
	}
}

func TestParseSSHConfigEndpointRejectsTemplateHost(t *testing.T) {
	if _, ok := parseSSHConfigEndpoint("hostname %h\nport 22\n"); ok {
		t.Fatal("template host should not be used as a network probe endpoint")
	}
}

func TestConnectRunsSSH(t *testing.T) {
	runner := &connectFakeRunner{}
	manager := Manager{
		Runner:          runner,
		connectionProbe: successfulConnectionProbe,
	}

	if err := manager.Connect(context.Background(), Entry{Name: "local", Target: "local"}); err != nil {
		t.Fatalf("Connect returned error: %v", err)
	}

	if runner.runName != "ssh" {
		t.Fatalf("runName = %q, want ssh", runner.runName)
	}
	if got := strings.Join(runner.runArgs, " "); got != "local" {
		t.Fatalf("runArgs = %q, want local", got)
	}
}

type connectFakeRunner struct {
	configOutput string
	runName      string
	runArgs      []string
}

func (r *connectFakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	r.runName = name
	r.runArgs = append([]string(nil), args...)
	return nil
}

func (r *connectFakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	if name == "ssh" && len(args) == 2 && args[0] == "-G" {
		return []byte(r.configOutput), nil
	}
	return nil, errors.New("unexpected output command")
}

func (r *connectFakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected output with input command")
}

func (r *connectFakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *connectFakeRunner) LookPath(name string) (string, error) {
	if name == "ssh" {
		return "/usr/bin/ssh", nil
	}
	return "", errors.New("not found")
}
