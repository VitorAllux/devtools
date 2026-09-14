package ports

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/desktop"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

type Manager struct {
	Config *config.Config
	Runner run.Runner
}

type Entry struct {
	Port    int
	PID     int
	Process string
	Address string
	Project string
	CWD     string
	Command string
	URL     string
}

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "help", "--help", "-h":
			showHelp(cfg)
			return nil
		default:
			return fmt.Errorf("unknown ports action: %s", args[0])
		}
	}
	return (&Manager{Config: cfg, Runner: runner}).Hub(ctx)
}

func (m *Manager) List(ctx context.Context) ([]Entry, error) {
	var entries []Entry
	var err error
	if _, lookErr := m.Runner.LookPath("ss"); lookErr == nil {
		out, outputErr := m.Runner.Output(ctx, "", "ss", "-ltnp", "-H")
		if outputErr == nil {
			entries = parseSS(string(out))
		} else {
			err = outputErr
		}
	}
	if len(entries) == 0 {
		if _, lookErr := m.Runner.LookPath("lsof"); lookErr == nil {
			out, outputErr := m.Runner.Output(ctx, "", "lsof", "-nP", "-iTCP", "-sTCP:LISTEN", "-Fpcn")
			if outputErr != nil {
				return nil, outputErr
			}
			entries = parseLSOF(string(out))
			err = nil
		}
	}
	if len(entries) == 0 && err != nil {
		return nil, err
	}
	for index := range entries {
		m.enrich(&entries[index])
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Port == entries[j].Port {
			return entries[i].PID < entries[j].PID
		}
		return entries[i].Port < entries[j].Port
	})
	return entries, nil
}

func (m *Manager) enrich(entry *Entry) {
	if entry.PID > 0 {
		entry.Command = procCommand(entry.PID)
		if replacement := betterProcessName(entry.Process, entry.Command); replacement != "" {
			entry.Process = replacement
		}
		if cwd, err := os.Readlink(filepath.Join("/proc", strconv.Itoa(entry.PID), "cwd")); err == nil {
			entry.CWD = cwd
			entry.Project = projectForCWD(m.Config, cwd)
		}
	}
	if entry.Project == "" {
		entry.Project = "system"
	}
	entry.URL = portURL(*entry)
}

func procCommand(pid int) string {
	if pid <= 0 {
		return ""
	}
	content, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cmdline"))
	if err != nil || len(content) == 0 {
		return ""
	}
	command := strings.ReplaceAll(string(content), "\x00", " ")
	return strings.Join(strings.Fields(command), " ")
}

func betterProcessName(current string, command string) string {
	current = strings.TrimSpace(current)
	command = strings.TrimSpace(command)
	if command == "" || (current != "" && !strings.EqualFold(current, "MainThread") && !strings.EqualFold(current, "unknown")) {
		return ""
	}
	lower := strings.ToLower(command)
	if strings.Contains(lower, ".vscode-server") && strings.Contains(lower, "extensionhost") {
		return "vscode-ext"
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return ""
	}
	name := filepath.Base(fields[0])
	if name == "node" && strings.Contains(lower, ".vscode-server") {
		return "vscode-node"
	}
	return name
}

var ssProcessPattern = regexp.MustCompile(`"([^"]+)".*pid=([0-9]+)`)

func parseSS(output string) []Entry {
	entries := []Entry{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		address := fields[3]
		port, ok := parseAddressPort(address)
		if !ok {
			continue
		}
		entry := Entry{Port: port, Address: address}
		if match := ssProcessPattern.FindStringSubmatch(line); len(match) == 3 {
			entry.Process = match[1]
			entry.PID, _ = strconv.Atoi(match[2])
		}
		entries = append(entries, entry)
	}
	return entries
}

func parseLSOF(output string) []Entry {
	entries := []Entry{}
	pid := 0
	process := ""
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(strings.TrimPrefix(line, "p"))
			process = ""
		case 'c':
			process = strings.TrimPrefix(line, "c")
		case 'n':
			address := strings.TrimPrefix(line, "n")
			port, ok := parseAddressPort(address)
			if !ok {
				continue
			}
			entries = append(entries, Entry{Port: port, PID: pid, Process: process, Address: address})
		}
	}
	return entries
}

func parseAddressPort(address string) (int, bool) {
	address = strings.TrimSpace(address)
	if index := strings.LastIndex(address, ":"); index >= 0 && index < len(address)-1 {
		value := address[index+1:]
		end := 0
		for end < len(value) && value[end] >= '0' && value[end] <= '9' {
			end++
		}
		value = value[:end]
		port, err := strconv.Atoi(value)
		return port, err == nil && port > 0
	}
	return 0, false
}

func projectForCWD(cfg *config.Config, cwd string) string {
	if cfg == nil {
		return ""
	}
	for _, project := range cfg.Project.Workspace.Projects {
		path := strings.TrimSpace(project.Path)
		if path != "" && pathContains(path, cwd) {
			return project.Name
		}
	}
	root := strings.TrimSpace(cfg.Project.Workspace.Root)
	if root == "" {
		return ""
	}
	rel, err := filepath.Rel(root, cwd)
	if err != nil || strings.HasPrefix(rel, "..") || rel == "." {
		return ""
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) >= 2 && strings.HasPrefix(parts[0], "workspace-") {
		return parts[1]
	}
	return ""
}

func pathContains(root string, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && (rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)))
}

func portURL(entry Entry) string {
	switch entry.Port {
	case 80:
		return "http://localhost"
	case 443:
		return "https://localhost"
	case 3000, 3001, 5173, 8000, 8080, 8081:
		return fmt.Sprintf("http://localhost:%d", entry.Port)
	}
	process := strings.ToLower(entry.Process)
	if strings.Contains(process, "node") || strings.Contains(process, "vite") || strings.Contains(process, "php") || strings.Contains(process, "artisan") {
		return fmt.Sprintf("http://localhost:%d", entry.Port)
	}
	return ""
}

func (m *Manager) open(ctx context.Context, entry Entry) error {
	if entry.URL == "" {
		return m.details(entry)
	}
	return desktop.OpenURL(ctx, m.Runner, entry.URL)
}

func (m *Manager) copy(ctx context.Context, entry Entry) error {
	if entry.URL == "" {
		return fmt.Errorf("port %d has no URL to copy", entry.Port)
	}
	if err := desktop.CopyToClipboard(ctx, m.Runner, entry.URL); err != nil {
		return err
	}
	ui.OK("Copied %s", entry.URL)
	return nil
}

func (m *Manager) kill(ctx context.Context, entry Entry) error {
	if entry.PID <= 0 {
		return fmt.Errorf("port %d has no detected PID", entry.Port)
	}
	if !ui.Confirm(fmt.Sprintf("Kill process %s (%d) on port %d?", displayProcess(entry), entry.PID, entry.Port)) {
		return nil
	}
	return m.Runner.Run(ctx, "", "kill", strconv.Itoa(entry.PID))
}

func (m *Manager) details(entry Entry) error {
	ui.Title("Port Details")
	fmt.Printf("  Port:    %d\n", entry.Port)
	fmt.Printf("  Bind:    %s\n", displayAddress(entry))
	fmt.Printf("  PID:     %s\n", displayPID(entry))
	fmt.Printf("  Process: %s\n", displayProcess(entry))
	fmt.Printf("  Project: %s\n", entry.Project)
	if entry.CWD != "" {
		fmt.Printf("  CWD:     %s\n", entry.CWD)
	}
	if entry.Command != "" {
		fmt.Printf("  Command: %s\n", entry.Command)
	}
	if entry.URL != "" {
		fmt.Printf("  URL:     %s\n", entry.URL)
	}
	return nil
}
