package secrets

import (
	"context"
	"fmt"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ssh"
	"github.com/VitorAllux/devtools/internal/ui"
)

type StatusItem struct {
	ID      string
	Label   string
	State   string
	Path    string
	Details string
}

func RunSecrets(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := Manager{Config: cfg, Runner: runner}
	if len(args) == 0 {
		return manager.Hub(ctx)
	}
	switch args[0] {
	case "help", "--help", "-h":
		showSecretsHelp(cfg)
		return nil
	case "status":
		manager.PrintStatus()
		return nil
	case "prepare":
		return manager.PrepareLocalSecrets(ctx)
	case "restore":
		return manager.RestoreSSHBackup(ctx)
	case "sync":
		return manager.SyncSSHBackup(ctx)
	default:
		return fmt.Errorf("unknown secrets action: %s", args[0])
	}
}

func (m Manager) Hub(ctx context.Context) error {
	hubError := ""
	for {
		items := m.StatusItems()
		if _, err := m.Runner.LookPath("fzf"); err == nil {
			keepOpen, nextError, err := m.fzfHub(ctx, items, hubError)
			if err != nil {
				return err
			}
			hubError = nextError
			if !keepOpen {
				return nil
			}
			continue
		}
		keepOpen, nextError, err := m.basicHub(ctx, items, hubError)
		if err != nil {
			return err
		}
		hubError = nextError
		if !keepOpen {
			return nil
		}
	}
}

func (m Manager) StatusItems() []StatusItem {
	return []StatusItem{
		fileStatus("age-key", "AGE private key", m.Config.AgeKeyFile, m.ageKeyExists(), "Private key used to decrypt local dvv backups."),
		fileStatus("age-recipients", "AGE recipients", m.Config.AgeRecipientsFile, fileHasContent(m.Config.AgeRecipientsFile), "Public recipients used when encrypting backups."),
		fileStatus("ssh-list", "SSH server list", m.Config.ServersFile, fileHasContent(m.Config.ServersFile), "Local SSH entries consumed by the SSH hub."),
		fileStatus("ssh-backup", "Encrypted SSH backup", m.Config.EncryptedServersFile, fileHasContent(m.Config.EncryptedServersFile), "Encrypted copy of the SSH server list."),
		{
			ID:      "bitwarden",
			Label:   "Bitwarden AGE item",
			State:   configuredState(m.Config.BitwardenAgeKeyItem),
			Path:    maskRef(m.Config.BitwardenAgeKeyItem),
			Details: "Optional Bitwarden item ID/name used to restore the AGE private key.",
		},
	}
}

func (m Manager) PrintStatus() {
	ui.Title("Secrets")
	for _, item := range m.StatusItems() {
		fmt.Printf("  %-20s %-12s %s\n", item.Label, item.State, item.Path)
	}
}

func (m Manager) PrepareLocalSecrets(ctx context.Context) error {
	return ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "preparing", Subject: "secrets", ShowResult: true, SuccessAction: "ready"}, func() error {
		if err := m.ensureLocalDirs(); err != nil {
			return err
		}
		if err := m.generateAgeKeyIfMissing(ctx); err != nil {
			return err
		}
		if publicKey := m.publicKeyFromPrivateKey(); publicKey != "" {
			if err := m.ensureRecipientsFileWithKey(publicKey); err != nil {
				return err
			}
		}
		return (ssh.Store{Path: m.Config.ServersFile}).Ensure()
	})
}

func (m Manager) RestoreSSHBackup(ctx context.Context) error {
	return m.Bootstrap(ctx, BootstrapOptions{ForceRestore: true, SkipSetup: true})
}

func (m Manager) SyncSSHBackup(ctx context.Context) error {
	return ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "syncing", Subject: "ssh backup", ShowResult: true, SuccessAction: "synced"}, func() error {
		if err := m.ensureLocalDirs(); err != nil {
			return err
		}
		if err := (ssh.Store{Path: m.Config.ServersFile}).Ensure(); err != nil {
			return err
		}
		return ssh.NewManager(m.Config, m.Runner).SyncBackup(ctx)
	})
}

func (m Manager) fzfHub(ctx context.Context, items []StatusItem, hubError string) (bool, string, error) {
	keys := m.Config.SecretsHubKeys()
	shortcuts := secretsHubShortcuts(keys)
	args := ui.FZFHub{
		Prompt:        ui.Crown("secrets") + ui.Muted("> "),
		BorderLabel:   "dvv secrets",
		HeaderLines:   secretsHeaderLines(hubError),
		Preview:       secretsPreviewCommand(shortcuts),
		PreviewLabel:  "secret panel",
		PreviewWindow: "right,42%,border-rounded,wrap",
		Shortcuts:     shortcuts,
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=6",
			"--nth=1,2,3,4,5,6",
			"--header-lines=1",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(secretRows(items)), "fzf", args...)
	if err != nil && len(output) == 0 {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	switch key {
	case keys.Prepare.FZFKey:
		if err := m.PrepareLocalSecrets(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Restore.FZFKey:
		if err := m.RestoreSSHBackup(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	case keys.Sync.FZFKey:
		if err := m.SyncSSHBackup(ctx); err != nil {
			return true, err.Error(), nil
		}
		return true, "", nil
	}
	raw := ui.FZFSelectedRaw(firstSelected(selected))
	if raw == "" {
		return false, "", nil
	}
	item, ok := findStatusItem(items, raw)
	if !ok {
		return true, "Selected secret item no longer exists", nil
	}
	fmt.Println(describeStatusItem(item))
	_, _ = ui.Prompt("Press Enter to return")
	return true, "", nil
}

func (m Manager) basicHub(ctx context.Context, items []StatusItem, hubError string) (bool, string, error) {
	keys := m.Config.SecretsHubKeys()
	if strings.TrimSpace(hubError) != "" {
		ui.Error("%s", hubError)
	}
	m.PrintStatus()
	fmt.Println()
	fmt.Printf("Commands: number details | %s prepare | %s restore | %s sync | q exits\n", keys.Prepare.Label, keys.Restore.Label, keys.Sync.Label)
	value, err := ui.Prompt("Secrets")
	if err != nil {
		return false, "", err
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "q") || strings.EqualFold(value, "quit") {
		return false, "", nil
	}
	switch {
	case matchesShortcut(value, keys.Prepare.FZFKey):
		return true, actionError(m.PrepareLocalSecrets(ctx)), nil
	case matchesShortcut(value, keys.Restore.FZFKey):
		return true, actionError(m.RestoreSSHBackup(ctx)), nil
	case matchesShortcut(value, keys.Sync.FZFKey):
		return true, actionError(m.SyncSSHBackup(ctx)), nil
	}
	index, ok := parseIndex(value, len(items))
	if !ok {
		return true, "Invalid secret selection: " + value, nil
	}
	fmt.Println(describeStatusItem(items[index]))
	_, _ = ui.Prompt("Press Enter to return")
	return true, "", nil
}

func fileStatus(id string, label string, path string, ok bool, details string) StatusItem {
	state := "missing"
	if ok {
		state = "ready"
	}
	return StatusItem{ID: id, Label: label, State: state, Path: path, Details: details}
}

func configuredState(value string) string {
	if strings.TrimSpace(value) == "" {
		return "not configured"
	}
	return "configured"
}

func maskRef(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<empty>"
	}
	return "<set>"
}

func secretRows(items []StatusItem) string {
	var builder strings.Builder
	builder.WriteString(secretLine("__dvv_header__", "", "", "", "", secretHeader()))
	builder.WriteByte('\n')
	for index, item := range items {
		builder.WriteString(secretLine(item.ID, item.Label, item.State, item.Path, item.Details, secretRow(index, item)))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func secretHeader() string {
	return fmt.Sprintf(" %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("SECRET", 22)),
		ui.Crown(fixedWidth("STATE", 14)),
	)
}

func secretRow(index int, item StatusItem) string {
	return fmt.Sprintf("%s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(item.Label, 22)),
		ui.Gold(fixedWidth(item.State, 14)),
	)
}

func secretLine(raw string, label string, state string, path string, details string, display string) string {
	return strings.Join([]string{
		cleanField(raw),
		cleanField(label),
		cleanField(state),
		cleanField(path),
		cleanField(details),
		display,
	}, "\t")
}

func secretsHubShortcuts(keys config.SecretsHubKeyBindings) []ui.FZFShortcut {
	return []ui.FZFShortcut{
		{Label: "Enter", Description: "details"},
		{Key: keys.Prepare.FZFKey, Label: keys.Prepare.Label, Description: "prepare local files"},
		{Key: keys.Restore.FZFKey, Label: keys.Restore.Label, Description: "restore backup"},
		{Key: keys.Sync.FZFKey, Label: keys.Sync.Label, Description: "sync backup"},
		{Label: "Esc", Description: "exit hub"},
	}
}

func secretsHeaderLines(message string) []string {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil
	}
	return []string{ui.Danger("error") + " " + ui.Danger(message)}
}

func secretsPreviewCommand(shortcuts []ui.FZFShortcut) string {
	commandDeck := ui.FZFPreviewCommandDeck(shortcuts)
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
id=$(printf "%s" "$line" | cut -f1)
name=$(printf "%s" "$line" | cut -f2)
state=$(printf "%s" "$line" | cut -f3)
path=$(printf "%s" "$line" | cut -f4)
details=$(printf "%s" "$line" | cut -f5)
print_commands() {
` + commandDeck + `
}
printf "%sSecret item%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "ID" "$dvv_reset" "$id"
printf "  %s%-8s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$name"
printf "  %s%-8s%s %s\n" "$dvv_label" "State" "$dvv_reset" "$state"
printf "  %s%-8s%s %s\n" "$dvv_label" "Path" "$dvv_reset" "$path"
printf "\n%sWhat it does%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$details" "$dvv_reset"
printf "\n%s--------------------------------%s\n" "$dvv_muted" "$dvv_reset"
printf "%sHub commands%s\n" "$dvv_heading" "$dvv_reset"
print_commands
' sh {}`
}

func describeStatusItem(item StatusItem) string {
	return strings.Join([]string{
		"ID: " + item.ID,
		"Name: " + item.Label,
		"State: " + item.State,
		"Path: " + item.Path,
		"Details: " + item.Details,
	}, "\n")
}

func findStatusItem(items []StatusItem, id string) (StatusItem, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return StatusItem{}, false
}

func actionError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
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
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		value = string(runes[:width-1]) + "."
	}
	return fmt.Sprintf("%-*s", width, value)
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
	if len(fzfKey) == 1 {
		return strings.EqualFold(input, fzfKey)
	}
	return false
}

func firstSelected(selected []string) string {
	if len(selected) == 0 {
		return ""
	}
	return selected[0]
}

func showSecretsHelp(cfg *config.Config) {
	keys := cfg.SecretsHubKeys()
	ui.Title("Secrets Hub")
	fmt.Printf("  %s dvv secrets\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv secrets", "Open the interactive local secrets hub")
	helpEntry("dvv bootstrap", "Run full setup and secrets bootstrap")
	fmt.Println()
	helpSection("Hub Shortcuts")
	helpEntry("Enter", "Show selected secret details")
	helpEntry(keys.Prepare.Label, "Prepare local AGE and SSH files")
	helpEntry(keys.Restore.Label, "Restore SSH list from encrypted backup")
	helpEntry(keys.Sync.Label, "Sync encrypted SSH backup from local SSH list")
	helpEntry("Esc", "Exit")
}
