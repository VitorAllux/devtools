package ports

import (
	"context"
	"fmt"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/ui"
)

func (m *Manager) Hub(ctx context.Context) error {
	for {
		entries, err := m.List(ctx)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			ui.Info("No listening TCP ports found")
			return nil
		}
		if _, err := m.Runner.LookPath("fzf"); err == nil {
			keepOpen, err := m.fzfHub(ctx, entries)
			if err != nil || !keepOpen {
				return err
			}
			continue
		}
		keepOpen, err := m.basicHub(ctx, entries)
		if err != nil || !keepOpen {
			return err
		}
	}
}

func (m *Manager) fzfHub(ctx context.Context, entries []Entry) (bool, error) {
	keys := m.Config.PortsHubKeys()
	shortcuts := portHubShortcuts(keys)
	args := ui.FZFHub{
		Prompt:        ui.Crown("ports") + ui.Muted("> "),
		BorderLabel:   "dvv ports",
		BorderTag:     "port manager",
		Preview:       portPreviewCommand(shortcuts),
		PreviewLabel:  "port panel",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: append(portRowArgs(),
			"--header-lines=1",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(portRows(entries)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	entry, ok := selectedPortEntry(entries, selected)
	if !ok {
		return false, nil
	}
	switch key {
	case keys.Kill.FZFKey:
		return true, m.kill(ctx, entry)
	case keys.Copy.FZFKey:
		return true, m.copy(ctx, entry)
	default:
		return true, m.open(ctx, entry)
	}
}

func (m *Manager) basicHub(ctx context.Context, entries []Entry) (bool, error) {
	keys := m.Config.PortsHubKeys()
	printPorts(entries)
	fmt.Printf("Commands: number opens/details | %s number kills | %s number copies URL | q exits\n", keys.Kill.Label, keys.Copy.Label)
	value, err := ui.Prompt("Port")
	if err != nil {
		return false, err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "q") || strings.EqualFold(value, "quit") {
		return false, nil
	}
	fields := strings.Fields(value)
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Kill.FZFKey) {
		index, ok := parseIndex(fields[1], len(entries))
		if !ok {
			return true, fmt.Errorf("invalid port selection: %s", fields[1])
		}
		return true, m.kill(ctx, entries[index])
	}
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Copy.FZFKey) {
		index, ok := parseIndex(fields[1], len(entries))
		if !ok {
			return true, fmt.Errorf("invalid port selection: %s", fields[1])
		}
		return true, m.copy(ctx, entries[index])
	}
	index, ok := parseIndex(value, len(entries))
	if !ok {
		return true, fmt.Errorf("invalid port selection: %s", value)
	}
	return true, m.open(ctx, entries[index])
}

func portHubShortcuts(keys config.PortsHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "open URL or details"},
		{Key: keys.Copy.FZFKey, Label: keys.Copy.Label, Description: "copy URL"},
		{Key: keys.Kill.FZFKey, Label: keys.Kill.Label, Description: "kill process"},
		{Label: "Esc", Description: "exit hub"},
	}
}

func portRows(entries []Entry) string {
	var builder strings.Builder
	builder.WriteString(portLine("__dvv_header__", "", "", "", "", "", "", "", "", portTableHeader()))
	builder.WriteByte('\n')
	for index, entry := range entries {
		builder.WriteString(portLine(portRaw(entry), fmt.Sprintf("%d", entry.Port), displayPID(entry), displayProcess(entry), entry.Project, entry.URL, displayAddress(entry), entry.CWD, entry.Command, portRow(index, entry)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func portRaw(entry Entry) string {
	return strings.Join([]string{fmt.Sprintf("%d", entry.Port), displayPID(entry), displayAddress(entry)}, ":")
}

func portLine(raw string, port string, pid string, process string, project string, url string, address string, cwd string, command string, display string) string {
	return strings.Join([]string{
		cleanField(raw),
		cleanField(port),
		cleanField(pid),
		cleanField(process),
		cleanField(project),
		cleanField(url),
		cleanField(address),
		cleanField(cwd),
		cleanField(command),
		display,
	}, "\t")
}

func portRowArgs() []string {
	return []string{
		"--delimiter=\t",
		"--with-nth=10",
		"--nth=1,2,3,4,5,6,7,8,9,10",
	}
}

func portTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("PORT", 6)),
		ui.Crown(fixedWidth("BIND", 17)),
		ui.Crown(fixedWidth("PID", 8)),
		ui.Crown(fixedWidth("PROCESS", 16)),
		ui.Crown(fixedWidth("PROJECT", 18)),
		ui.Crown("URL"),
	)
}

func portRow(index int, entry Entry) string {
	return fmt.Sprintf("%s  %s  %s  %s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Gold(fixedWidth(fmt.Sprintf("%d", entry.Port), 6)),
		ui.Muted(fixedWidth(displayAddress(entry), 17)),
		ui.Muted(fixedWidth(displayPID(entry), 8)),
		ui.Accent(fixedWidth(displayProcess(entry), 16)),
		ui.Gold(fixedWidth(entry.Project, 18)),
		ui.Muted(entry.URL),
	)
}

func selectedPortEntry(entries []Entry, selected []string) (Entry, bool) {
	if len(selected) == 0 {
		return Entry{}, false
	}
	raw, _, _ := strings.Cut(selected[0], "\t")
	raw = strings.TrimSpace(raw)
	for _, entry := range entries {
		if portRaw(entry) == raw {
			return entry, true
		}
	}
	return Entry{}, false
}

func portPreviewCommand(shortcuts []ui.FZFShortcut) string {
	commandDeck := ui.FZFPreviewCommandDeck(shortcuts)
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
port=$(printf "%s" "$line" | cut -f2)
pid=$(printf "%s" "$line" | cut -f3)
process=$(printf "%s" "$line" | cut -f4)
project=$(printf "%s" "$line" | cut -f5)
url=$(printf "%s" "$line" | cut -f6)
address=$(printf "%s" "$line" | cut -f7)
cwd=$(printf "%s" "$line" | cut -f8)
command=$(printf "%s" "$line" | cut -f9)
print_commands() {
` + commandDeck + `
}
printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
print_commands
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
printf "%sPort%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Port" "$dvv_reset" "$port"
printf "  %s%-8s%s %s\n" "$dvv_label" "Bind" "$dvv_reset" "$address"
printf "  %s%-8s%s %s\n" "$dvv_label" "PID" "$dvv_reset" "$pid"
printf "  %s%-8s%s %s\n" "$dvv_label" "Process" "$dvv_reset" "$process"
printf "  %s%-8s%s %s\n" "$dvv_label" "Project" "$dvv_reset" "$project"
if [ -n "$url" ]; then
  printf "  %s%-8s%s %s\n" "$dvv_label" "URL" "$dvv_reset" "$url"
fi
if [ -n "$cwd" ]; then
  printf "  %s%-8s%s %s\n" "$dvv_label" "CWD" "$dvv_reset" "$cwd"
fi
if [ -n "$command" ]; then
  printf "\n%sCommand%s\n" "$dvv_heading" "$dvv_reset"
  printf "  %s%s%s\n" "$dvv_muted" "$command" "$dvv_reset"
fi
printf "  %s%-8s%s %s\n" "$dvv_label" "Target" "$dvv_reset" "$raw"
' sh {}`
}

func printPorts(entries []Entry) {
	ui.Title("Port Manager")
	for index, entry := range entries {
		fmt.Printf("  %2d. %-6d %-17s %-8s %-16s %-18s %s\n", index+1, entry.Port, displayAddress(entry), displayPID(entry), displayProcess(entry), entry.Project, entry.URL)
	}
}

func displayPID(entry Entry) string {
	if entry.PID <= 0 {
		return "?"
	}
	return fmt.Sprintf("%d", entry.PID)
}

func displayProcess(entry Entry) string {
	if strings.TrimSpace(entry.Process) == "" {
		return "unknown"
	}
	return entry.Process
}

func displayAddress(entry Entry) string {
	if strings.TrimSpace(entry.Address) == "" {
		return "?"
	}
	return entry.Address
}

func cleanField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func fixedWidth(value string, width int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) >= width {
		return string(runes[:width])
	}
	return value + strings.Repeat(" ", width-len(runes))
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

func matchesShortcut(input string, fzfKey string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	fzfKey = strings.ToLower(strings.TrimSpace(fzfKey))
	if input == "" || fzfKey == "" {
		return false
	}
	if input == fzfKey {
		return true
	}
	if strings.HasPrefix(fzfKey, "shift+") && len(input) == 1 {
		return input == strings.TrimPrefix(fzfKey, "shift+")
	}
	return false
}

func showHelp(cfg *config.Config) {
	keys := (&config.Config{Project: config.DefaultProjectConfig()}).PortsHubKeys()
	if cfg != nil {
		keys = cfg.PortsHubKeys()
	}
	ui.Title("Port Manager")
	fmt.Printf("  %s dvv ports\n\n", ui.Bold("Usage:"))
	fmt.Printf("  %s\n", ui.Cyan("Hub:"))
	fmt.Printf("    %-28s %s\n", ui.Bold("dvv ports"), "Open local listening ports")
	fmt.Println()
	fmt.Printf("  %s\n", ui.Cyan("Hub Shortcuts:"))
	fmt.Printf("    %-28s %s\n", ui.Bold("Enter"), "Open URL or show details")
	fmt.Printf("    %-28s %s\n", ui.Bold(keys.Copy.Label), "Copy URL")
	fmt.Printf("    %-28s %s\n", ui.Bold(keys.Kill.Label), "Kill process after confirmation")
	fmt.Printf("    %-28s %s\n", ui.Bold("Esc"), "Exit")
}
