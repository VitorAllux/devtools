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

	got, err := (Manager{Config: testConfig(), Runner: runner, OS: "linux"}).Resources(context.Background())
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

func TestResourcesDetectsBrewServicesOnDarwin(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &fakeRunner{
		paths: map[string]bool{"brew": true},
		outputs: map[string][]byte{
			"brew services list": []byte("Name          Status  User File\nmysql         started user ~/Library/LaunchAgents/homebrew.mxcl.mysql.plist\nredis         none\n"),
		},
	}

	got, err := (Manager{Config: testConfig(), Runner: runner, OS: "darwin"}).Resources(context.Background())
	if err != nil {
		t.Fatalf("Resources returned error: %v", err)
	}

	mysql, ok := findByID(got, "service:mysql")
	if !ok {
		t.Fatalf("mysql brew service missing: %#v", got)
	}
	if mysql.Manager != "brew" || mysql.State != "running" || mysql.Details != "user" {
		t.Fatalf("mysql brew service = %#v", mysql)
	}
	redis, ok := findByID(got, "service:redis")
	if !ok || redis.State != "stopped" {
		t.Fatalf("redis brew service = %#v ok=%v", redis, ok)
	}
	if runner.lookedUp("service") || runner.lookedUp("systemctl") {
		t.Fatalf("macOS resources should not probe Linux service managers: %#v", runner.lookups)
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

func TestRunActionUsesBrewServicesCommand(t *testing.T) {
	runner := &fakeRunner{paths: map[string]bool{"brew": true}}
	resource := Resource{
		ID:        "service:mysql",
		Kind:      "Service",
		Name:      "mysql",
		State:     "stopped",
		Available: true,
		Manager:   "brew",
		Actions:   []string{"start", "stop", "restart"},
		Target:    "mysql",
	}

	result, err := (Manager{Config: testConfig(), Runner: runner, OS: "darwin"}).RunAction(context.Background(), resource, "start", false)
	if err != nil {
		t.Fatalf("RunAction returned error: %v", err)
	}
	if !result.Success || !runner.hasRun("brew services start mysql") {
		t.Fatalf("result = %#v runs = %#v", result, runner.runs)
	}
}

func TestLogsCommandUsesResourceManager(t *testing.T) {
	cfg := testConfig()
	cfg.Project.Resources.Logs.Tail = 50
	manager := Manager{Config: cfg, Runner: &fakeRunner{
		paths: map[string]bool{"docker": true, "journalctl": true},
		outputs: map[string][]byte{
			"docker compose version": []byte("Docker Compose version v2.20.0\n"),
		},
	}}

	container, err := manager.logsCommand(context.Background(), Resource{
		Kind:      "Container",
		Name:      "web",
		Available: true,
		Manager:   "docker",
		Target:    "web",
	})
	if err != nil {
		t.Fatalf("container logsCommand returned error: %v", err)
	}
	if got := strings.Join(container, " "); got != "docker logs --tail 50 -f web" {
		t.Fatalf("container logs command = %q", got)
	}

	compose, err := manager.logsCommand(context.Background(), Resource{
		Kind:         "Compose",
		Name:         "saas",
		Available:    true,
		Manager:      "docker compose",
		Target:       "saas",
		ComposeFiles: []string{"docker-compose.yml", "docker-compose.override.yml"},
	})
	if err != nil {
		t.Fatalf("compose logsCommand returned error: %v", err)
	}
	if got := strings.Join(compose, " "); got != "docker compose -f docker-compose.yml -f docker-compose.override.yml -p saas logs --tail 50 -f" {
		t.Fatalf("compose logs command = %q", got)
	}

	service, err := manager.logsCommand(context.Background(), Resource{
		Kind:      "Service",
		Name:      "mysql",
		Available: true,
		Manager:   "systemctl",
		Target:    "mysql",
	})
	if err != nil {
		t.Fatalf("service logsCommand returned error: %v", err)
	}
	if got := strings.Join(service, " "); got != "journalctl -fu mysql.service -n 50" {
		t.Fatalf("service logs command = %q", got)
	}
}

func TestOpenLogsLaunchesTerminal(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &fakeRunner{paths: map[string]bool{"x-terminal-emulator": true}}
	manager := Manager{Config: testConfig(), Runner: runner, OS: "linux"}

	err := manager.OpenLogs(context.Background(), Resource{
		Kind:      "Container",
		Name:      "web",
		Available: true,
		Manager:   "docker",
		Target:    "web",
	})
	if err != nil {
		t.Fatalf("OpenLogs returned error: %v", err)
	}
	if !runner.hasStart("x-terminal-emulator -e docker logs --tail 200 -f web") {
		t.Fatalf("terminal start missing: %#v", runner.starts)
	}
}

func TestCommandDetailsAndActionUseDetectedResource(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &fakeRunner{
		paths: map[string]bool{"service": true},
		outputs: map[string][]byte{
			"service --status-all": []byte(" [ - ]  mysql\n"),
			"service mysql status": []byte("mysql is stopped\n"),
		},
	}
	manager := Manager{Config: testConfig(), Runner: runner, OS: "linux"}

	if err := manager.CommandDetails(context.Background(), []string{"service:mysql"}); err != nil {
		t.Fatalf("CommandDetails returned error: %v", err)
	}
	if err := manager.CommandAction(context.Background(), "start", []string{"service:mysql", "--sudo"}); err != nil {
		t.Fatalf("CommandAction returned error: %v", err)
	}
	if !runner.hasRun("service mysql start") && !runner.hasRun("sudo service mysql start") {
		t.Fatalf("runs = %#v, want service start with host-appropriate sudo", runner.runs)
	}
}

func TestResourceFormattingHelpers(t *testing.T) {
	resource := Resource{
		ID:        "container:web",
		Kind:      "Container",
		Name:      "web",
		State:     "running",
		Available: true,
		Manager:   "docker",
		Details:   "nginx | Up",
		Actions:   []string{"start"},
		Target:    "web",
	}
	if text := describe(resource); !strings.Contains(text, "container:web") || !strings.Contains(text, "Actions") {
		t.Fatalf("describe = %q", text)
	}
	if command := formatCommand([]string{"docker", "compose", "-f", "a b.yml", "up"}); command != "'docker' 'compose' '-f' 'a b.yml' 'up'" {
		t.Fatalf("formatCommand = %q", command)
	}
	if actionGerund("restart") != "restarting" || actionDone("stop") != "stopped" || actionDone("other") != "done" {
		t.Fatal("action label helpers returned unexpected values")
	}
	if actionTitle("") != "" || actionDoneTitle("restart") != "Restarted" {
		t.Fatal("action title helpers returned unexpected values")
	}
	if shellQuote("a'b") != "'a'\\''b'" {
		t.Fatal("shellQuote did not escape single quotes")
	}
}

func TestResourceHelperEdges(t *testing.T) {
	if got := dockerUnavailableDetail("permission denied while connecting"); !strings.Contains(got, "current user") {
		t.Fatalf("permission detail = %q", got)
	}
	if got := dockerUnavailableDetail("Cannot connect to the Docker daemon"); !strings.Contains(got, "not reachable") {
		t.Fatalf("daemon detail = %q", got)
	}
	if got := dockerUnavailableDetail("Docker Desktop WSL 2 distro integration disabled"); !strings.Contains(got, "WSL integration") {
		t.Fatalf("wsl detail = %q", got)
	}
	if got := dockerStateFromStatus("Created"); got != "stopped" {
		t.Fatalf("created state = %q", got)
	}
	if got := dockerStateFromStatus("Paused 2 minutes"); got != "paused" {
		t.Fatalf("paused state = %q", got)
	}
	if got := dockerStateFromStatus(""); got != "unknown" {
		t.Fatalf("empty state = %q", got)
	}
	if resourceKindRank("Service") >= resourceKindRank("Other") {
		t.Fatal("known resources should sort before unknown resources")
	}
	if got := firstLine("\x00\r\n  first line  \nsecond", "fallback"); got != "first line" {
		t.Fatalf("firstLine = %q", got)
	}
	if got := firstLine("", "fallback"); got != "fallback" {
		t.Fatalf("firstLine fallback = %q", got)
	}
	if got := firstLine(strings.Repeat("x", 200), "fallback"); len(got) != 180 {
		t.Fatalf("firstLine should cap long lines at 180 chars, got %d", len(got))
	}
	if got := firstNonEmpty(" ", "\tvalue\n"); got != "value" {
		t.Fatalf("firstNonEmpty = %q", got)
	}
	if got := safeIDPart(" mysql@8.0!* "); got != "mysql_8.0" {
		t.Fatalf("safeIDPart = %q", got)
	}
	if got := safeIDPart("!!!"); got != "unknown" {
		t.Fatalf("safeIDPart fallback = %q", got)
	}
	if !matchesShortcut("s", "alt-s") || !matchesShortcut("ctrl-r", "ctrl-r") || matchesShortcut("", "alt-s") {
		t.Fatal("matchesShortcut returned unexpected values")
	}
	if index, ok := parseIndex("2", 3); !ok || index != 1 {
		t.Fatalf("parseIndex = %d ok=%v", index, ok)
	}
	for _, value := range []string{"0", "4", "x"} {
		if _, ok := parseIndex(value, 3); ok {
			t.Fatalf("parseIndex(%q) should be invalid", value)
		}
	}
	if firstSelected(nil) != "" || firstSelected([]string{"alpha", "beta"}) != "alpha" {
		t.Fatal("firstSelected returned unexpected values")
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

func TestParseBrewServices(t *testing.T) {
	services := parseBrewServices([]byte("Name Status User File\npostgresql@14 stopped\nmysql started alice ~/plist\n"))
	if len(services) != 2 {
		t.Fatalf("services = %#v", services)
	}
	if services[0].Name != "postgresql@14" || services[0].Status != "stopped" || services[0].User != "" {
		t.Fatalf("first service = %#v", services[0])
	}
	if state := brewServiceState("started"); state != "running" {
		t.Fatalf("started state = %q", state)
	}
	if state := brewServiceState("none"); state != "stopped" {
		t.Fatalf("none state = %q", state)
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
	visible := strings.Split(strings.Split(strings.TrimSpace(rows), "\n")[1], "\t")[6]
	if strings.Contains(visible, "nginx | Up") {
		t.Fatalf("visible resource row should keep details in preview only: %q", visible)
	}
	if !strings.Contains(visible, "docker") {
		t.Fatalf("visible resource row should keep manager column: %q", visible)
	}
}

func testConfig() *config.Config {
	return &config.Config{Project: config.DefaultProjectConfig()}
}

type fakeRunner struct {
	paths   map[string]bool
	outputs map[string][]byte
	runs    []string
	starts  []string
	lookups []string
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

func (r *fakeRunner) Start(_ context.Context, _ string, name string, args ...string) error {
	r.starts = append(r.starts, commandKey(name, args...))
	return nil
}

func (r *fakeRunner) LookPath(name string) (string, error) {
	r.lookups = append(r.lookups, name)
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

func (r *fakeRunner) lookedUp(command string) bool {
	for _, lookup := range r.lookups {
		if lookup == command {
			return true
		}
	}
	return false
}

func (r *fakeRunner) hasStart(command string) bool {
	for _, start := range r.starts {
		if start == command {
			return true
		}
	}
	return false
}

func commandKey(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), " ")
}
