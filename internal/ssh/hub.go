package ssh

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/ui"
)

func (m *Manager) Hub(ctx context.Context) error {
	hubError := ""
	for {
		entries, err := m.Store.Entries()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return m.emptyHub(ctx)
		}

		if _, err := m.Runner.LookPath("fzf"); err == nil {
			keepOpen, nextHubError, err := m.fzfHub(ctx, entries, hubError)
			if err != nil {
				return err
			}
			hubError = nextHubError
			if !keepOpen {
				return nil
			}
			continue
		}

		keepOpen, nextHubError, err := m.basicHub(ctx, entries, hubError)
		if err != nil {
			return err
		}
		hubError = nextHubError
		if !keepOpen {
			return nil
		}
	}
}

func (m *Manager) emptyHub(ctx context.Context) error {
	ui.Warn("No SSH entries found")
	if !ui.Confirm("Add a new SSH entry now?") {
		return nil
	}
	name, err := ui.Prompt("SSH entry name")
	if err != nil {
		return err
	}
	target, err := ui.Prompt("SSH target, for example user@example.com")
	if err != nil {
		return err
	}
	if err := m.Store.Add(name, target); err != nil {
		return err
	}
	ui.OK("Added SSH entry %s", strings.TrimSpace(name))
	m.syncAndReport(ctx)
	return nil
}

func (m *Manager) fzfHub(ctx context.Context, entries []Entry, hubError string) (bool, string, error) {
	keys := m.Config.SSHHubKeys()
	shortcuts := sshHubShortcuts(keys)
	args := ui.FZFHub{
		Prompt:        ui.Crown("search") + ui.Muted("> "),
		BorderLabel:   "dvv ssh",
		HeaderLines:   sshHubHeaderLines(hubError),
		Preview:       sshPreviewCommand(shortcuts),
		PreviewLabel:  "hub panel",
		PreviewWindow: "right,34%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: append(ui.FZFHiddenRowArgs(),
			"--header-lines=1",
		),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(styledEntriesInput(entries)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, "", nil
	}

	key, selection := parseFZFExpectOutput(string(output))
	selection = ui.FZFSelectedRaw(selection)
	switch key {
	case keys.Add.FZFKey:
		if err := m.addFromPrompt(ctx); err != nil {
			return true, hubErrorMessage(err), nil
		}
		return true, "", nil
	case keys.Remove.FZFKey:
		if selection == "" {
			ui.Warn("No SSH entry selected")
			return true, "", nil
		}
		entry, ok := findEntryByRaw(entries, selection)
		if !ok {
			return true, hubErrorMessage(fmt.Errorf("selected SSH entry no longer exists")), nil
		}
		if !ui.Confirm(fmt.Sprintf("Remove %s?", entry.Name)) {
			return true, "", nil
		}
		removed, err := m.Store.RemoveRaw([]string{entry.Raw})
		if err != nil {
			return true, hubErrorMessage(err), nil
		}
		if removed > 0 {
			ui.OK("Removed SSH entry %s", entry.Name)
			m.syncAndReport(ctx)
		}
		return true, "", nil
	case keys.NewTerminal.FZFKey:
		if selection == "" {
			ui.Warn("No SSH entry selected")
			return true, "", nil
		}
		entry, ok := findEntryByRaw(entries, selection)
		if !ok {
			return true, hubErrorMessage(fmt.Errorf("selected SSH entry no longer exists")), nil
		}
		if err := m.OpenInNewTerminal(ctx, entry); err != nil {
			return true, hubErrorMessage(err), nil
		}
		return true, "", nil
	default:
		if selection == "" {
			return false, "", nil
		}
		entry, ok := findEntryByRaw(entries, selection)
		if !ok {
			return true, hubErrorMessage(fmt.Errorf("selected SSH entry no longer exists")), nil
		}
		if err := m.OpenInNewTerminal(ctx, entry); err != nil {
			return true, hubErrorMessage(err), nil
		}
		return false, "", nil
	}
}

func (m *Manager) basicHub(ctx context.Context, entries []Entry, hubError string) (bool, string, error) {
	keys := m.Config.SSHHubKeys()
	if strings.TrimSpace(hubError) != "" {
		ui.Error("%s", hubError)
	}
	printEntryList(entries)
	fmt.Println()
	fmt.Printf("Commands: number opens terminal | %s adds | %s number removes | %s number opens terminal | q exits\n", keys.Add.Label, keys.Remove.Label, keys.NewTerminal.Label)
	value, err := ui.Prompt("SSH")
	if err != nil {
		return false, "", err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "q") || strings.EqualFold(value, "quit") {
		return false, "", nil
	}
	if matchesShortcut(value, keys.Add.FZFKey) {
		if err := m.addFromPrompt(ctx); err != nil {
			return true, hubErrorMessage(err), nil
		}
		return true, "", nil
	}

	fields := strings.Fields(value)
	if len(fields) == 2 && matchesShortcut(fields[0], keys.Remove.FZFKey) {
		index, ok := parseEntryIndex(fields[1], len(entries))
		if !ok {
			return true, hubErrorMessage(fmt.Errorf("invalid SSH entry selection: %s", fields[1])), nil
		}
		if ui.Confirm(fmt.Sprintf("Remove %s?", entries[index].Name)) {
			removed, err := m.Store.RemoveRaw([]string{entries[index].Raw})
			if err != nil {
				return true, hubErrorMessage(err), nil
			}
			if removed > 0 {
				ui.OK("Removed SSH entry %s", entries[index].Name)
				m.syncAndReport(ctx)
			}
		}
		return true, "", nil
	}
	if len(fields) == 2 && matchesShortcut(fields[0], keys.NewTerminal.FZFKey) {
		index, ok := parseEntryIndex(fields[1], len(entries))
		if !ok {
			return true, hubErrorMessage(fmt.Errorf("invalid SSH entry selection: %s", fields[1])), nil
		}
		if err := m.OpenInNewTerminal(ctx, entries[index]); err != nil {
			return true, hubErrorMessage(err), nil
		}
		return true, "", nil
	}

	index, ok := parseEntryIndex(value, len(entries))
	if !ok {
		return true, hubErrorMessage(fmt.Errorf("invalid SSH entry selection: %s", value)), nil
	}
	if err := m.OpenInNewTerminal(ctx, entries[index]); err != nil {
		return true, hubErrorMessage(err), nil
	}
	return false, "", nil
}

func (m *Manager) addFromPrompt(ctx context.Context) error {
	name, err := ui.Prompt("SSH entry name")
	if err != nil {
		return err
	}
	target, err := ui.Prompt("SSH target, for example user@example.com")
	if err != nil {
		return err
	}
	if err := m.Store.Add(name, target); err != nil {
		return err
	}
	ui.OK("Added SSH entry %s", strings.TrimSpace(name))
	m.syncAndReport(ctx)
	return nil
}

func styledEntriesInput(entries []Entry) string {
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(sshTableHeader()))
	builder.WriteByte('\n')
	for index, entry := range entries {
		builder.WriteString(ui.FZFHiddenRow(entry.Raw, styledEntryRow(index, entry)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func styledEntryRow(index int, entry Entry) string {
	user, host := splitSSHTarget(entry.Target)
	return fmt.Sprintf("%s  %s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(entry.Name, 24)),
		ui.Gold(fixedWidth(user, 12)),
		ui.Muted(host),
	)
}

func splitSSHTarget(target string) (string, string) {
	target = strings.TrimSpace(target)
	user, host, ok := strings.Cut(target, "@")
	if !ok {
		return "-", target
	}
	return user, host
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

func sshTableHeader() string {
	return fmt.Sprintf(" %s  %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("TARGET", 24)),
		ui.Crown(fixedWidth("USER", 12)),
		ui.Crown("HOST"),
	)
}

func sshHubShortcuts(keys config.SSHHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "open terminal"},
		{Key: keys.Add.FZFKey, Label: keys.Add.Label, Description: "add SSH entry"},
		{Key: keys.Remove.FZFKey, Label: keys.Remove.Label, Description: "remove selected"},
		{Key: keys.NewTerminal.FZFKey, Label: keys.NewTerminal.Label, Description: "open terminal tab"},
		{Label: "Esc", Description: "exit hub"},
	}
}

func sshRemoveShortcuts() []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Tab", Description: "mark entry"},
		{Label: "Enter", Description: "remove selected"},
		{Label: "Esc", Description: "cancel"},
	}
}

func parseFZFExpectOutput(output string) (string, string) {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 {
		return "", ""
	}
	if len(lines) == 1 {
		return "", strings.TrimSpace(lines[0])
	}
	return strings.TrimSpace(lines[0]), strings.TrimSpace(lines[1])
}

func nonEmptyLines(output string) []string {
	lines := strings.Split(output, "\n")
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		line = ui.FZFSelectedRaw(line)
		if line != "" {
			values = append(values, line)
		}
	}
	return values
}

func findEntryByRaw(entries []Entry, raw string) (Entry, bool) {
	raw = ui.FZFSelectedRaw(raw)
	for _, entry := range entries {
		if entry.Raw == raw {
			return entry, true
		}
	}
	return Entry{}, false
}

func printEntryList(entries []Entry) {
	ui.Title("SSH Hub")
	for index, entry := range entries {
		fmt.Printf("  %2d. %-20s %s\n", index+1, entry.Name, ui.Dim(entry.Target))
	}
}

func parseEntryIndex(value string, length int) (int, bool) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	index := number - 1
	return index, index >= 0 && index < length
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

func sshHubHeaderLines(message string) []string {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	return []string{ui.Danger("error") + " " + ui.Danger(message)}
}

func hubErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func sshPreviewCommand(shortcuts []ui.FZFShortcut) string {
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
raw=$(printf "%s" "$line" | cut -f1)
name=${raw%% *}
target=${raw#* }
user=${target%@*}
host=${target#*@}
if [ "$user" = "$target" ]; then
  user="-"
  host="$target"
fi
printf "%sTarget profile%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-7s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$name"
printf "  %s%-7s%s %s\n" "$dvv_label" "User" "$dvv_reset" "$user"
printf "  %s%-7s%s %s\n" "$dvv_label" "Host" "$dvv_reset" "$host"
printf "  %s%-7s%s %s\n" "$dvv_label" "Target" "$dvv_reset" "$target"
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
while [ "$#" -gt 1 ]; do
  printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "$1" "$dvv_reset" "$dvv_muted" "$2" "$dvv_reset"
  shift 2
done
printf "\n%sConfigured in dvv.config.json.%s\n" "$dvv_muted" "$dvv_reset"
' sh {} ` + strings.Join(shortcutArgs, " ")
}
