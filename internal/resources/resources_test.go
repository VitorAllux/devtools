package resources

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

func TestResourcesDetectsServiceEntries(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &fakeRunner{
		paths: map[string]bool{"service": true},
		outputs: map[string][]byte{
			"service --status-all":        []byte(" [ + ]  redis-server\n [ - ]  mysql\n"),
			"service redis-server status": []byte("redis-server is running\n"),
			"service mysql status":        []byte("mysql is stopped\n"),
		},
	}

	got, err := (Manager{Config: testConfig(), Runner: runner}).Resources(context.Background())
	if err != nil {
		t.Fatalf("Resources returned error: %v", err)
	}

	redis, ok := findByID(got, "service:redis-server")
	if !ok {
		t.Fatalf("redis service missing: %#v", got)
	}
	if redis.State != "running" || redis.Manager != "service" {
		t.Fatalf("redis resource = %#v", redis)
	}
	mysql, ok := findByID(got, "service:mysql")
	if !ok || mysql.State != "stopped" {
		t.Fatalf("mysql resource = %#v ok=%v", mysql, ok)
	}
}

func TestResourcesDetectsDockerContainerAndComposeProject(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &fakeRunner{
		paths: map[string]bool{"docker": true},
		outputs: map[string][]byte{
			"docker version --format {{.Server.Version}}":                        []byte("24.0.0\n"),
			"docker ps -a --format {{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}": []byte("abc123\tweb\tnginx:alpine\tUp 2 minutes\n"),
			"docker compose version":                                             []byte("Docker Compose version v2.20.0\n"),
			"docker compose ls --format json":                                    []byte(`[{"Name":"saas","Status":"running(1)","ConfigFiles":"/tmp/docker-compose.yml"}]`),
		},
	}

	got, err := (Manager{Config: testConfig(), Runner: runner}).Resources(context.Background())
	if err != nil {
		t.Fatalf("Resources returned error: %v", err)
	}

	container, ok := findByID(got, "container:web")
	if !ok {
		t.Fatalf("container missing: %#v", got)
	}
	if container.State != "running" || container.Target != "web" {
		t.Fatalf("container = %#v", container)
	}
	compose, ok := findByID(got, "compose:saas")
	if !ok {
		t.Fatalf("compose project missing: %#v", got)
	}
	if compose.Manager != "docker compose" || len(compose.ComposeFiles) != 1 {
		t.Fatalf("compose = %#v", compose)
	}
}

func TestRunActionUsesDockerCommand(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"docker": true}}
	resource := Resource{
		ID:        "container:web",
		Kind:      "Container",
		Name:      "web",
		State:     "stopped",
		Available: true,
		Manager:   "docker",
		Actions:   []string{"start", "stop", "restart"},
		Target:    "web",
	}

	result, err := (Manager{Config: testConfig(), Runner: runner}).RunAction(context.Background(), resource, "restart", false)
	if err != nil {
		t.Fatalf("RunAction returned error: %v", err)
	}
	if !result.Success {
		t.Fatalf("result = %#v", result)
	}
	if !runner.hasRun("docker restart web") {
		t.Fatalf("runs = %#v", runner.runs)
	}
}

func TestParseComposeProjectsSupportsJSONObjectPerLine(t *testing.T) {
	got := parseComposeProjects([]byte("{\"Name\":\"api\",\"Status\":\"running\",\"ConfigFiles\":\"a.yml,b.yml\"}\n"))
	if len(got) != 1 || got[0].Name != "api" {
		t.Fatalf("projects = %#v", got)
	}
	files := splitComposeFiles(got[0].ConfigFiles)
	if len(files) != 2 || files[0] != "a.yml" || files[1] != "b.yml" {
		t.Fatalf("files = %#v", files)
	}
}

func TestResourceRowsKeepRawIDHidden(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	rows := resourceRows([]Resource{{
		ID:      "container:web",
		Kind:    "Container",
		Name:    "web",
		State:   "running",
		Manager: "docker",
		Details: "nginx | Up",
	}})
	if !strings.Contains(rows, "container:web\tContainer\tweb\trunning\tdocker\tnginx | Up\t") {
		t.Fatalf("rows = %q", rows)
	}
}

func testConfig() *config.Config {
	return &config.Config{Project: config.DefaultProjectConfig()}
}

type fakeRunner struct {
	paths   map[string]bool
	outputs map[string][]byte
	runs    []string
}

func (r *fakeRunner) Run(_ context.Context, _ string, name string, args ...string) error {
	r.runs = append(r.runs, commandKey(name, args...))
	return nil
}

func (r *fakeRunner) Output(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	key := commandKey(name, args...)
	if output, ok := r.outputs[key]; ok {
		return output, nil
	}
	return nil, errors.New("unexpected output command: " + key)
}

func (r *fakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	return nil, errors.New("unexpected fzf command")
}

func (r *fakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	if r.paths[name] {
		return "/usr/bin/" + name, nil
	}
	return "", errors.New("not found")
}

func (r *fakeRunner) hasRun(command string) bool {
	for _, run := range r.runs {
		if run == command {
			return true
		}
	}
	return false
}

func commandKey(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), " ")
}
