package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

var serviceStatusPattern = regexp.MustCompile(`^\s*\[\s*([+\-?])\s*\]\s+(.+?)\s*$`)
var unsafeIDPartPattern = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

var serviceDefinitions = []struct {
	Name  string
	Label string
}{
	{"redis-server", "Redis"},
	{"postgresql", "PostgreSQL"},
	{"mysql", "MySQL"},
	{"mariadb", "MariaDB"},
	{"docker", "Docker daemon"},
}

var actionNames = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
}

type Resource struct {
	ID           string
	Kind         string
	Name         string
	State        string
	Available    bool
	Manager      string
	Details      string
	Actions      []string
	RequiresSudo bool
	Target       string
	ComposeFiles []string
}

type ActionResult struct {
	Success          bool
	Message          string
	Command          []string
	RequiresTerminal bool
}

type Manager struct {
	Config *config.Config
	Runner run.Runner
}

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := Manager{Config: cfg, Runner: runner}
	if len(args) == 0 {
		return manager.Hub(ctx)
	}

	switch args[0] {
	case "help", "--help", "-h":
		showHelp(cfg)
		return nil
	case "details":
		return manager.CommandDetails(ctx, args[1:])
	case "start", "stop", "restart":
		return manager.CommandAction(ctx, args[0], args[1:])
	default:
		return fmt.Errorf("unknown resources action: %s", args[0])
	}
}

func (m Manager) Hub(ctx context.Context) error {
	hubError := ""
	for {
		resources, err := m.Load(ctx)
		if err != nil {
			return err
		}
		if len(resources) == 0 {
			if strings.TrimSpace(hubError) != "" {
				ui.Error("%s", hubError)
			}
			ui.Info("No local resources detected.")
			return nil
		}
		if _, err := m.Runner.LookPath("fzf"); err == nil {
			keepOpen, nextError, err := m.fzfHub(ctx, resources, hubError)
			if err != nil {
				return err
			}
			hubError = nextError
			if !keepOpen {
				return nil
			}
			continue
		}

		keepOpen, nextError, err := m.basicHub(ctx, resources, hubError)
		if err != nil {
			return err
		}
		hubError = nextError
		if !keepOpen {
			return nil
		}
	}
}

func (m Manager) Load(ctx context.Context) ([]Resource, error) {
	var resources []Resource
	err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "scanning", Subject: "resources"}, func() error {
		var loadErr error
		resources, loadErr = m.Resources(ctx)
		return loadErr
	})
	return resources, err
}

func (m Manager) Resources(ctx context.Context) ([]Resource, error) {
	serviceEntries := m.serviceStatusEntries(ctx)
	hasSystemctl := m.systemctlUsable(ctx)
	resources := make([]Resource, 0, len(serviceDefinitions)+8)

	for _, definition := range serviceDefinitions {
		resource, ok := m.detectService(ctx, definition.Name, definition.Label, serviceEntries, hasSystemctl)
		if ok {
			resources = append(resources, resource)
		}
	}

	serviceIDs := map[string]bool{}
	for _, resource := range resources {
		serviceIDs[resource.ID] = true
	}
	for _, resource := range m.detectDockerResources(ctx) {
		if resource.ID == "docker:daemon" && serviceIDs["service:docker"] {
			continue
		}
		resources = append(resources, resource)
	}

	sort.SliceStable(resources, func(i, j int) bool {
		if resources[i].Kind == resources[j].Kind {
			return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
		}
		return resourceKindRank(resources[i].Kind) < resourceKindRank(resources[j].Kind)
	})
	return resources, nil
}

func (m Manager) CommandList(ctx context.Context) error {
	resources, err := m.Load(ctx)
	if err != nil {
		return err
	}
	if len(resources) == 0 {
		ui.Info("No local resources detected.")
		return nil
	}
	fmt.Println(formatTable(resources))
	return nil
}

func (m Manager) CommandDetails(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: dvv resources details <resource-id>")
	}
	resource, ok, err := m.Find(ctx, args[0])
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("resource not found: %s", args[0])
	}
	fmt.Println(describe(resource))
	return nil
}

func (m Manager) CommandAction(ctx context.Context, action string, args []string) error {
	useSudo := false
	var resourceID string
	for _, arg := range args {
		switch arg {
		case "--sudo":
			useSudo = true
		default:
			if resourceID == "" {
				resourceID = arg
			} else {
				return fmt.Errorf("unexpected resources argument: %s", arg)
			}
		}
	}
	if strings.TrimSpace(resourceID) == "" {
		return fmt.Errorf("usage: dvv resources %s <resource-id> [--sudo]", action)
	}
	resource, ok, err := m.Find(ctx, resourceID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("resource not found: %s", resourceID)
	}
	var result ActionResult
	err = ui.RunWithRoyalLoader(ui.LoaderOptions{Action: actionGerund(action), Subject: resource.Name, ShowResult: true, SuccessAction: actionDone(action)}, func() error {
		var runErr error
		result, runErr = m.RunAction(ctx, resource, action, useSudo)
		return runErr
	})
	if err != nil {
		return err
	}
	if result.RequiresTerminal && len(result.Command) > 0 {
		ui.Warn("%s: %s", result.Message, formatCommand(result.Command))
		return nil
	}
	if !result.Success {
		return fmt.Errorf("%s", result.Message)
	}
	ui.OK("%s", result.Message)
	return nil
}

func (m Manager) Find(ctx context.Context, resourceID string) (Resource, bool, error) {
	resources, err := m.Resources(ctx)
	if err != nil {
		return Resource{}, false, err
	}
	for _, resource := range resources {
		if resource.ID == resourceID {
			return resource, true, nil
		}
	}
	return Resource{}, false, nil
}

func (m Manager) RunAction(ctx context.Context, resource Resource, action string, useSudo bool) (ActionResult, error) {
	if !actionNames[action] {
		return ActionResult{}, fmt.Errorf("unknown resource action: %s", action)
	}
	if !contains(resource.Actions, action) {
		return ActionResult{}, fmt.Errorf("%s does not support %s", resource.Name, action)
	}
	if !resource.Available {
		return ActionResult{}, fmt.Errorf("%s is unavailable: %s", resource.Name, resource.Details)
	}
	command := m.actionCommand(ctx, resource, action, useSudo)
	if len(command) == 0 {
		return ActionResult{}, fmt.Errorf("no command available to %s %s", action, resource.Name)
	}
	if resource.RequiresSudo && currentUserNeedsSudo() && !useSudo {
		return ActionResult{
			Success:          false,
			Message:          fmt.Sprintf("%s for %s requires sudo", actionTitle(action), resource.Name),
			Command:          m.actionCommand(ctx, resource, action, true),
			RequiresTerminal: true,
		}, nil
	}
	if err := m.Runner.Run(ctx, "", command[0], command[1:]...); err != nil {
		return ActionResult{}, err
	}
	return ActionResult{
		Success: true,
		Message: fmt.Sprintf("%s %s", actionDoneTitle(action), resource.Name),
		Command: command,
	}, nil
}

func (m Manager) fzfHub(ctx context.Context, resources []Resource, hubError string) (bool, string, error) {
	keys := m.Config.ResourcesHubKeys()
	shortcuts := resourceHubShortcuts(keys)
	args := ui.FZFHub{
		Prompt:        ui.Crown("resources") + ui.Muted("> "),
		BorderLabel:   "dvv resources",
		HeaderLines:   resourceHubHeaderLines(hubError),
		Preview:       resourcePreviewCommand(shortcuts),
		PreviewLabel:  "resource panel",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=7",
			"--nth=1,2,3,4,5,6,7",
			"--header-lines=1",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(resourceRows(resources)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, "", nil
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	resourceID := ui.FZFSelectedRaw(firstSelected(selected))
	if resourceID == "" {
		return false, "", nil
	}
	resource, ok := findByID(resources, resourceID)
	if !ok {
		return true, "Selected resource no longer exists", nil
	}
	switch key {
	case keys.Start.FZFKey:
		return true, m.runHubAction(ctx, resource, "start"), nil
	case keys.Restart.FZFKey:
		return true, m.runHubAction(ctx, resource, "restart"), nil
	case keys.Stop.FZFKey:
		return true, m.runHubAction(ctx, resource, "stop"), nil
	default:
		fmt.Println(describe(resource))
		_, _ = ui.Prompt("Press Enter to return")
		return true, "", nil
	}
}

func (m Manager) runHubAction(ctx context.Context, resource Resource, action string) string {
	var result ActionResult
	err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: actionGerund(action), Subject: resource.Name, ShowResult: true, SuccessAction: actionDone(action)}, func() error {
		var runErr error
		result, runErr = m.RunAction(ctx, resource, action, true)
		return runErr
	})
	if err != nil {
		return err.Error()
	}
	if result.RequiresTerminal && len(result.Command) > 0 {
		return result.Message + ": " + formatCommand(result.Command)
	}
	ui.OK("%s", result.Message)
	return ""
}

func (m Manager) basicHub(ctx context.Context, resources []Resource, hubError string) (bool, string, error) {
	keys := m.Config.ResourcesHubKeys()
	if strings.TrimSpace(hubError) != "" {
		ui.Error("%s", hubError)
	}
	ui.Title("Resources Hub")
	fmt.Println(formatTable(resources))
	fmt.Println()
	fmt.Printf("Commands: number details | %s number start | %s number restart | %s number stop | q exits\n", keys.Start.Label, keys.Restart.Label, keys.Stop.Label)
	value, err := ui.Prompt("Resources")
	if err != nil {
		return false, "", err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "q") || strings.EqualFold(value, "quit") {
		return false, "", nil
	}
	fields := strings.Fields(value)
	if len(fields) == 2 {
		action := ""
		switch {
		case matchesShortcut(fields[0], keys.Start.FZFKey):
			action = "start"
		case matchesShortcut(fields[0], keys.Restart.FZFKey):
			action = "restart"
		case matchesShortcut(fields[0], keys.Stop.FZFKey):
			action = "stop"
		}
		if action != "" {
			index, ok := parseIndex(fields[1], len(resources))
			if !ok {
				return true, "Invalid resource selection: " + fields[1], nil
			}
			return true, m.runHubAction(ctx, resources[index], action), nil
		}
	}
	index, ok := parseIndex(value, len(resources))
	if !ok {
		return true, "Invalid resource selection: " + value, nil
	}
	fmt.Println(describe(resources[index]))
	_, _ = ui.Prompt("Press Enter to return")
	return true, "", nil
}

func (m Manager) serviceStatusEntries(ctx context.Context) map[string]string {
	entries := map[string]string{}
	if _, err := m.Runner.LookPath("service"); err != nil {
		return entries
	}
	output, _ := m.outputWithTimeout(ctx, 8*time.Second, "", "service", "--status-all")
	for _, line := range strings.Split(cleanText(string(output)), "\n") {
		match := serviceStatusPattern.FindStringSubmatch(line)
		if len(match) == 3 {
			entries[strings.TrimSpace(match[2])] = strings.TrimSpace(match[1])
		}
	}
	return entries
}

func (m Manager) systemctlUsable(ctx context.Context) bool {
	if _, err := m.Runner.LookPath("systemctl"); err != nil {
		return false
	}
	_, err := m.outputWithTimeout(ctx, 3*time.Second, "", "systemctl", "list-units", "--type=service", "--no-pager", "--plain")
	return err == nil
}

func (m Manager) systemctlServiceExists(ctx context.Context, serviceName string) bool {
	unit := serviceName + ".service"
	output, err := m.outputWithTimeout(ctx, 3*time.Second, "", "systemctl", "list-unit-files", unit, "--no-pager", "--plain")
	return err == nil && strings.Contains(string(output), unit)
}

func (m Manager) detectService(ctx context.Context, serviceName string, label string, serviceEntries map[string]string, hasSystemctl bool) (Resource, bool) {
	needsSudo := currentUserNeedsSudo()
	if hasSystemctl && m.systemctlServiceExists(ctx, serviceName) {
		unit := serviceName + ".service"
		output, _ := m.outputWithTimeout(ctx, 3*time.Second, "", "systemctl", "is-active", unit)
		return Resource{
			ID:           "service:" + serviceName,
			Kind:         "Service",
			Name:         label,
			State:        firstLine(string(output), "unknown"),
			Available:    true,
			Manager:      "systemctl",
			Details:      "systemd unit: " + unit,
			Actions:      resourceActions(),
			RequiresSudo: needsSudo,
			Target:       serviceName,
		}, true
	}

	marker, ok := serviceEntries[serviceName]
	if !ok {
		return Resource{}, false
	}
	output, err := m.outputWithTimeout(ctx, 5*time.Second, "", "service", serviceName, "status")
	state := "unknown"
	if marker == "+" {
		state = "running"
	} else if marker == "-" {
		state = "stopped"
	} else if err == nil {
		state = "running"
	}
	details := firstLine(string(output), "service entry: "+serviceName)
	if strings.Contains(strings.ToLower(details), "failed to connect to bus") {
		details = "service entry: " + serviceName
	}
	return Resource{
		ID:           "service:" + serviceName,
		Kind:         "Service",
		Name:         label,
		State:        state,
		Available:    true,
		Manager:      "service",
		Details:      details,
		Actions:      resourceActions(),
		RequiresSudo: needsSudo,
		Target:       serviceName,
	}, true
}

func (m Manager) detectDockerResources(ctx context.Context) []Resource {
	if _, err := m.Runner.LookPath("docker"); err != nil {
		return nil
	}

	version, err := m.outputWithTimeout(ctx, 6*time.Second, "", "docker", "version", "--format", "{{.Server.Version}}")
	if err != nil {
		return []Resource{{
			ID:        "docker:daemon",
			Kind:      "Docker",
			Name:      "Docker daemon",
			State:     "unavailable",
			Available: false,
			Manager:   "docker",
			Details:   dockerUnavailableDetail(string(version)),
			Target:    "docker",
		}}
	}

	resources := []Resource{{
		ID:        "docker:daemon",
		Kind:      "Docker",
		Name:      "Docker daemon",
		State:     "running",
		Available: true,
		Manager:   "docker",
		Details:   "Docker server " + firstLine(string(version), "available"),
		Target:    "docker",
	}}
	resources = append(resources, m.detectDockerContainers(ctx)...)
	resources = append(resources, m.detectComposeProjects(ctx)...)
	return resources
}

func (m Manager) detectDockerContainers(ctx context.Context) []Resource {
	output, err := m.outputWithTimeout(ctx, 8*time.Second, "", "docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	if err != nil {
		return nil
	}
	var resources []Resource
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		containerID := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		image := strings.TrimSpace(parts[2])
		status := strings.TrimSpace(parts[3])
		if name == "" {
			continue
		}
		target := name
		if target == "" {
			target = containerID
		}
		resources = append(resources, Resource{
			ID:        "container:" + safeIDPart(name),
			Kind:      "Container",
			Name:      name,
			State:     dockerStateFromStatus(status),
			Available: true,
			Manager:   "docker",
			Details:   image + " | " + status,
			Actions:   resourceActions(),
			Target:    target,
		})
	}
	return resources
}

func (m Manager) detectComposeProjects(ctx context.Context) []Resource {
	base := m.composeCommandBase(ctx)
	if len(base) == 0 {
		return nil
	}
	output, err := m.outputWithTimeout(ctx, 8*time.Second, "", base[0], append(base[1:], "ls", "--format", "json")...)
	if err != nil {
		return nil
	}
	projects := parseComposeProjects(output)
	resources := make([]Resource, 0, len(projects))
	for _, project := range projects {
		name := strings.TrimSpace(project.Name)
		if name == "" {
			continue
		}
		configFiles := splitComposeFiles(project.ConfigFiles)
		resources = append(resources, Resource{
			ID:           "compose:" + safeIDPart(name),
			Kind:         "Compose",
			Name:         name,
			State:        firstNonEmpty(project.Status, "unknown"),
			Available:    true,
			Manager:      strings.Join(base, " "),
			Details:      firstNonEmpty(project.ConfigFiles, "Docker Compose project"),
			Actions:      resourceActions(),
			Target:       name,
			ComposeFiles: configFiles,
		})
	}
	return resources
}

func (m Manager) composeCommandBase(ctx context.Context) []string {
	if _, err := m.Runner.LookPath("docker"); err == nil {
		if _, err := m.outputWithTimeout(ctx, 5*time.Second, "", "docker", "compose", "version"); err == nil {
			return []string{"docker", "compose"}
		}
	}
	if _, err := m.Runner.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}
	}
	return nil
}

func (m Manager) actionCommand(ctx context.Context, resource Resource, action string, useSudo bool) []string {
	if !actionNames[action] {
		return nil
	}
	prefix := []string{}
	if useSudo && resource.RequiresSudo && currentUserNeedsSudo() {
		prefix = append(prefix, "sudo")
	}
	switch resource.Kind {
	case "Service":
		if resource.Manager == "systemctl" {
			return append(prefix, "systemctl", action, resource.Target+".service")
		}
		return append(prefix, "service", resource.Target, action)
	case "Container":
		return []string{"docker", action, resource.Target}
	case "Compose":
		base := m.composeCommandBase(ctx)
		if len(base) == 0 {
			return nil
		}
		command := append([]string{}, base...)
		for _, file := range resource.ComposeFiles {
			command = append(command, "-f", file)
		}
		command = append(command, "-p", resource.Target, action)
		return command
	default:
		return nil
	}
}

func (m Manager) outputWithTimeout(ctx context.Context, timeout time.Duration, dir string, name string, args ...string) ([]byte, error) {
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return m.Runner.Output(runCtx, dir, name, args...)
}

func formatTable(resources []Resource) string {
	rows := [][]string{{"KIND", "NAME", "STATE", "MANAGER", "DETAILS"}}
	for _, resource := range resources {
		rows = append(rows, []string{resource.Kind, resource.Name, resource.State, resource.Manager, resource.Details})
	}
	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for index, value := range row {
			if len(value) > widths[index] {
				widths[index] = len(value)
			}
		}
	}
	lines := make([]string, 0, len(rows)+1)
	for rowIndex, row := range rows {
		parts := make([]string, len(row))
		for index, value := range row {
			parts[index] = value + strings.Repeat(" ", widths[index]-len(value))
		}
		lines = append(lines, strings.TrimRight(strings.Join(parts, "  "), " "))
		if rowIndex == 0 {
			separators := make([]string, len(widths))
			for index, width := range widths {
				separators[index] = strings.Repeat("-", width)
			}
			lines = append(lines, strings.TrimRight(strings.Join(separators, "  "), " "))
		}
	}
	return strings.Join(lines, "\n")
}

func resourceRows(resources []Resource) string {
	var builder strings.Builder
	builder.WriteString(joinResourceFields("__dvv_header__", "", "", "", "", "", resourceTableHeader()))
	builder.WriteByte('\n')
	for index, resource := range resources {
		builder.WriteString(joinResourceFields(
			resource.ID,
			resource.Kind,
			resource.Name,
			resource.State,
			resource.Manager,
			resource.Details,
			resourceRow(index, resource),
		))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func resourceRow(index int, resource Resource) string {
	return fmt.Sprintf("%s  %s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(resource.Kind, 12)),
		ui.Accent(fixedWidth(resource.Name, 28)),
		ui.Gold(fixedWidth(resource.State, 14)),
		ui.Muted(fixedWidth(resource.Manager, 14)+" "+resource.Details),
	)
}

func resourceTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("KIND", 12)),
		ui.Crown(fixedWidth("NAME", 28)),
		ui.Crown(fixedWidth("STATE", 14)),
		ui.Crown("MANAGER / DETAILS"),
	)
}

func joinResourceFields(fields ...string) string {
	cleaned := make([]string, len(fields))
	for index, field := range fields {
		field = strings.ReplaceAll(field, "\t", " ")
		field = strings.ReplaceAll(field, "\r", " ")
		field = strings.ReplaceAll(field, "\n", " ")
		cleaned[index] = field
	}
	return strings.Join(cleaned, "\t")
}

func describe(resource Resource) string {
	actions := "none"
	if len(resource.Actions) > 0 {
		actions = strings.Join(resource.Actions, ", ")
	}
	available := "no"
	if resource.Available {
		available = "yes"
	}
	sudo := "no"
	if resource.RequiresSudo {
		sudo = "yes"
	}
	return strings.Join([]string{
		"ID: " + resource.ID,
		"Kind: " + resource.Kind,
		"Name: " + resource.Name,
		"State: " + resource.State,
		"Available: " + available,
		"Manager: " + resource.Manager,
		"Actions: " + actions,
		"Requires sudo: " + sudo,
		"Details: " + resource.Details,
	}, "\n")
}

func resourceHubShortcuts(keys config.ResourcesHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "details"},
		{Key: keys.Start.FZFKey, Label: keys.Start.Label, Description: "start"},
		{Key: keys.Restart.FZFKey, Label: keys.Restart.Label, Description: "restart"},
		{Key: keys.Stop.FZFKey, Label: keys.Stop.Label, Description: "stop"},
		{Label: "Esc", Description: "exit hub"},
	}
}

func resourceHubHeaderLines(message string) []string {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	return []string{ui.Danger("error") + " " + ui.Danger(message)}
}

func resourcePreviewCommand(shortcuts []ui.FZFShortcut) string {
	shortcutArgs := make([]string, 0, len(shortcuts)*2)
	for _, shortcut := range shortcuts {
		label := strings.TrimSpace(shortcut.Label)
		if label == "" {
			label = strings.TrimSpace(shortcut.Key)
		}
		description := strings.TrimSpace(shortcut.Description)
		if label == "" || description == "" {
			continue
		}
		shortcutArgs = append(shortcutArgs, shellQuote(label), shellQuote(description))
	}

	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
shift
id=$(printf "%s" "$line" | cut -f1)
kind=$(printf "%s" "$line" | cut -f2)
name=$(printf "%s" "$line" | cut -f3)
state=$(printf "%s" "$line" | cut -f4)
manager=$(printf "%s" "$line" | cut -f5)
details=$(printf "%s" "$line" | cut -f6)
printf "%sResource profile%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "ID" "$dvv_reset" "$id"
printf "  %s%-8s%s %s\n" "$dvv_label" "Kind" "$dvv_reset" "$kind"
printf "  %s%-8s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$name"
printf "  %s%-8s%s %s\n" "$dvv_label" "State" "$dvv_reset" "$state"
printf "  %s%-8s%s %s\n" "$dvv_label" "Manager" "$dvv_reset" "$manager"
printf "  %s%-8s%s %s\n" "$dvv_label" "Details" "$dvv_reset" "$details"
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
while [ "$#" -gt 1 ]; do
  printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "$1" "$dvv_reset" "$dvv_muted" "$2" "$dvv_reset"
  shift 2
done
printf "\n%sConfigured in dvv.config.json.%s\n" "$dvv_muted" "$dvv_reset"
' sh {} ` + strings.Join(shortcutArgs, " ")
}

type composeProject struct {
	Name        string
	Status      string
	ConfigFiles string
}

func parseComposeProjects(output []byte) []composeProject {
	content := strings.TrimSpace(string(output))
	if content == "" {
		return nil
	}
	var array []composeProject
	if err := json.Unmarshal([]byte(content), &array); err == nil {
		return array
	}
	var one composeProject
	if err := json.Unmarshal([]byte(content), &one); err == nil && strings.TrimSpace(one.Name) != "" {
		return []composeProject{one}
	}
	var rows []composeProject
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row composeProject
		if err := json.Unmarshal([]byte(line), &row); err == nil {
			rows = append(rows, row)
		}
	}
	return rows
}

func splitComposeFiles(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' }) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func dockerUnavailableDetail(output string) string {
	lower := strings.ToLower(output)
	switch {
	case strings.Contains(lower, "wsl 2 distro") || strings.Contains(lower, "wsl integration"):
		return "Docker CLI found, but Docker Desktop WSL integration is not enabled for this distro."
	case strings.Contains(lower, "permission denied"):
		return "Docker CLI found, but the current user cannot access the Docker daemon."
	case strings.Contains(lower, "cannot connect") || strings.Contains(lower, "is the docker daemon running"):
		return "Docker CLI found, but the Docker daemon is not reachable."
	default:
		return firstLine(output, "Docker CLI found, but Docker is unavailable.")
	}
}

func dockerStateFromStatus(status string) string {
	lower := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.HasPrefix(lower, "up"):
		return "running"
	case strings.Contains(lower, "exited") || strings.Contains(lower, "created"):
		return "stopped"
	case lower != "":
		return strings.Fields(lower)[0]
	default:
		return "unknown"
	}
}

func resourceActions() []string {
	return []string{"start", "stop", "restart"}
}

func resourceKindRank(kind string) int {
	switch kind {
	case "Service":
		return 10
	case "Docker":
		return 20
	case "Container":
		return 30
	case "Compose":
		return 40
	default:
		return 100
	}
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\x00", "")
	text = strings.ReplaceAll(text, "\r", "")
	return strings.TrimSpace(text)
}

func firstLine(text string, fallback string) string {
	for _, line := range strings.Split(cleanText(text), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 180 {
				return line[:180]
			}
			return line
		}
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func safeIDPart(value string) string {
	value = unsafeIDPartPattern.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_")
	if value == "" {
		return "unknown"
	}
	return value
}

func currentUserNeedsSudo() bool {
	return runtime.GOOS != "windows" && os.Geteuid() != 0
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func actionTitle(action string) string {
	if action == "" {
		return ""
	}
	return strings.ToUpper(action[:1]) + action[1:]
}

func actionGerund(action string) string {
	switch action {
	case "start":
		return "starting"
	case "stop":
		return "stopping"
	case "restart":
		return "restarting"
	default:
		return action
	}
}

func actionDone(action string) string {
	switch action {
	case "start":
		return "started"
	case "stop":
		return "stopped"
	case "restart":
		return "restarted"
	default:
		return "done"
	}
}

func actionDoneTitle(action string) string {
	return actionTitle(actionDone(action))
}

func formatCommand(command []string) string {
	parts := make([]string, len(command))
	for index, part := range command {
		parts[index] = shellQuote(part)
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func fixedWidth(value string, width int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		value = string(runes[:width-1]) + "."
	}
	return fmt.Sprintf("%-*s", width, value)
}

func matchesShortcut(input string, fzfKey string) bool {
	input = strings.TrimSpace(input)
	fzfKey = strings.TrimSpace(fzfKey)
	if input == "" || fzfKey == "" {
		return false
	}
	if strings.EqualFold(input, fzfKey) {
		return true
	}
	if strings.HasPrefix(fzfKey, "alt-") || strings.HasPrefix(fzfKey, "ctrl-") {
		return strings.EqualFold(input, strings.TrimPrefix(strings.TrimPrefix(fzfKey, "alt-"), "ctrl-"))
	}
	return false
}

func parseIndex(value string, length int) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	number := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
		number = number*10 + int(r-'0')
	}
	index := number - 1
	return index, index >= 0 && index < length
}

func findByID(resources []Resource, id string) (Resource, bool) {
	for _, resource := range resources {
		if resource.ID == id {
			return resource, true
		}
	}
	return Resource{}, false
}

func firstSelected(selected []string) string {
	if len(selected) == 0 {
		return ""
	}
	return selected[0]
}

func showHelp(cfg *config.Config) {
	keys := cfg.ResourcesHubKeys()
	ui.Title("Resources Hub")
	fmt.Printf("  %s dvv resources\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv resources", "Open the interactive local resources hub")
	fmt.Println()
	helpSection("Hub Shortcuts")
	helpEntry("Enter", "Show selected resource details")
	helpEntry(keys.Start.Label, "Start selected resource")
	helpEntry(keys.Restart.Label, "Restart selected resource")
	helpEntry(keys.Stop.Label, "Stop selected resource")
	helpEntry("Esc", "Exit")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-30s %s\n", ui.Bold(command), description)
}
