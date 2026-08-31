package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/setup"
	"github.com/VitorAllux/devtools/internal/ssh"
	"github.com/VitorAllux/devtools/internal/ui"
)

type BootstrapOptions struct {
	ForceRestore bool
	SkipSetup    bool
}

type Manager struct {
	Config *config.Config
	Runner run.Runner
}

type interactiveOutputRunner interface {
	InteractiveOutput(ctx context.Context, dir string, name string, args ...string) ([]byte, error)
}

func RunBootstrap(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	if len(args) > 0 && isHelpArg(args[0]) {
		showHelp()
		return nil
	}
	options, err := parseBootstrapOptions(args)
	if err != nil {
		return err
	}
	return Manager{Config: cfg, Runner: runner}.Bootstrap(ctx, options)
}

func (m Manager) Bootstrap(ctx context.Context, options BootstrapOptions) error {
	ui.Title("Bootstrap")
	if !options.SkipSetup {
		ui.Info("Running shell integration setup first")
		if err := (setup.Manager{Config: m.Config, Runner: m.Runner}).Setup(ctx); err != nil {
			return err
		}
	}

	if err := m.ensureLocalDirs(); err != nil {
		return err
	}
	if migrated, err := m.migrateLegacyServersFile(); err != nil {
		return err
	} else if migrated {
		ui.OK("Migrated legacy SSH list to %s", m.Config.ServersFile)
	}

	if fileHasContent(m.Config.EncryptedServersFile) {
		if !m.ageKeyExists() {
			ui.Info("Restoring AGE key from Bitwarden if configured")
			if restored, err := m.restoreAgeKeyFromBitwarden(ctx); err == nil && restored {
				ui.OK("AGE private key restored from Bitwarden")
			} else {
				if err != nil {
					ui.Warn("Could not restore AGE key from Bitwarden: %v", err)
				} else {
					ui.Warn("Could not restore AGE key from Bitwarden")
				}
				ui.Warn("Set DVV_BW_AGE_KEY_ITEM or DEVT_BW_AGE_KEY_ITEM in %s", m.Config.ConfigFile)
			}
		}
		if m.ageKeyExists() {
			publicKey := m.publicKeyFromPrivateKey()
			if publicKey != "" {
				if err := m.ensureRecipientsFileWithKey(publicKey); err != nil {
					return err
				}
			}
			shouldRestore := options.ForceRestore
			hasEntries, err := m.serversFileHasEntries()
			if err != nil {
				return err
			}
			if !hasEntries {
				shouldRestore = true
			}
			if shouldRestore {
				if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "decrypting", Subject: "ssh backup", ShowResult: true}, func() error {
					return ssh.NewManager(m.Config, m.Runner).RestoreBackup(ctx)
				}); err != nil {
					ui.Warn("Encrypted SSH backup exists, but decryption failed: %v", err)
				} else if hasEntries, _ := m.serversFileHasEntries(); hasEntries {
					ui.OK("Restored SSH servers from encrypted backup")
				} else {
					ui.Warn("Encrypted backup decrypted, but it has no server entries")
				}
			} else {
				ui.Info("Keeping current local SSH list; use --force to overwrite")
			}
		}
	} else {
		if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "preparing", Subject: "AGE key", ShowResult: true}, func() error {
			return m.generateAgeKeyIfMissing(ctx)
		}); err != nil {
			ui.Warn("Could not generate AGE key automatically. Install age and rerun `dvv bootstrap`.")
		} else if m.ageKeyExists() {
			ui.OK("AGE key ready at %s", m.Config.AgeKeyFile)
			publicKey := m.publicKeyFromPrivateKey()
			if publicKey != "" {
				if err := m.ensureRecipientsFileWithKey(publicKey); err != nil {
					return err
				}
				ui.OK("Recipient list ready at %s", m.Config.AgeRecipientsFile)
			}
		}
	}

	if err := (ssh.Store{Path: m.Config.ServersFile}).Ensure(); err != nil {
		return err
	}

	err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "encrypting", Subject: "ssh backup", ShowResult: true}, func() error {
		return ssh.NewManager(m.Config, m.Runner).SyncBackup(ctx)
	})
	if err != nil {
		if ssh.IsSkippedBackup(err) {
			hasEntries, _ := m.serversFileHasEntries()
			if hasEntries {
				ui.Warn("%s", err)
			} else {
				ui.Info("Skipped encrypted backup update because there are no SSH server entries yet")
			}
		} else {
			ui.Warn("Could not update encrypted SSH backup: %v", err)
		}
	} else {
		ui.OK("Encrypted SSH backup updated at %s", m.Config.EncryptedServersFile)
	}

	ui.Info("Bootstrap complete")
	ui.Info("For a new machine: bw login && dvv bootstrap --force")
	return nil
}

func parseBootstrapOptions(args []string) (BootstrapOptions, error) {
	options := BootstrapOptions{}
	for _, arg := range args {
		switch arg {
		case "--force":
			options.ForceRestore = true
		case "--skip-setup":
			options.SkipSetup = true
		default:
			return options, fmt.Errorf("unknown bootstrap option: %s", arg)
		}
	}
	return options, nil
}

func (m Manager) ensureLocalDirs() error {
	for _, dir := range []string{
		m.Config.ConfigDir,
		filepath.Dir(m.Config.ServersFile),
		filepath.Dir(m.Config.AgeKeyFile),
		filepath.Dir(m.Config.AgeRecipientsFile),
		filepath.Dir(m.Config.EncryptedServersFile),
	} {
		if strings.TrimSpace(dir) == "" || dir == "." {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		_ = os.Chmod(dir, 0o700)
	}
	return nil
}

func (m Manager) migrateLegacyServersFile() (bool, error) {
	hasEntries, err := m.serversFileHasEntries()
	if err != nil {
		return false, err
	}
	if hasEntries {
		return false, nil
	}
	legacy := filepath.Join(m.Config.RootDir, "config", "servers.list")
	if !fileHasContent(legacy) {
		return false, nil
	}
	content, err := os.ReadFile(legacy)
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(m.Config.ServersFile), 0o700); err != nil {
		return false, err
	}
	if err := os.WriteFile(m.Config.ServersFile, content, 0o600); err != nil {
		return false, err
	}
	return true, os.Chmod(m.Config.ServersFile, 0o600)
}

func (m Manager) ageKeyExists() bool {
	content, err := os.ReadFile(m.Config.AgeKeyFile)
	return err == nil && strings.Contains(string(content), "AGE-SECRET-KEY-")
}

func (m Manager) generateAgeKeyIfMissing(ctx context.Context) error {
	if m.ageKeyExists() {
		_ = os.Chmod(m.Config.AgeKeyFile, 0o600)
		return nil
	}
	if _, err := m.Runner.LookPath("age-keygen"); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(m.Config.AgeKeyFile), 0o700); err != nil {
		return err
	}
	if err := run.Quiet(ctx, m.Runner, "", "age-keygen", "-o", m.Config.AgeKeyFile); err != nil {
		return err
	}
	return os.Chmod(m.Config.AgeKeyFile, 0o600)
}

func (m Manager) publicKeyFromPrivateKey() string {
	content, err := os.ReadFile(m.Config.AgeKeyFile)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# public key: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# public key: "))
		}
	}
	return ""
}

func (m Manager) ensureRecipientsFileWithKey(publicKey string) error {
	if strings.TrimSpace(publicKey) == "" {
		return fmt.Errorf("AGE public key is empty")
	}
	if err := os.MkdirAll(filepath.Dir(m.Config.AgeRecipientsFile), 0o700); err != nil {
		return err
	}
	existing, _ := os.ReadFile(m.Config.AgeRecipientsFile)
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == publicKey {
			return os.Chmod(m.Config.AgeRecipientsFile, 0o600)
		}
	}
	file, err := os.OpenFile(m.Config.AgeRecipientsFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		if _, err := file.WriteString("\n"); err != nil {
			return err
		}
	}
	if _, err := file.WriteString(publicKey + "\n"); err != nil {
		return err
	}
	return os.Chmod(m.Config.AgeRecipientsFile, 0o600)
}

func (m Manager) restoreAgeKeyFromBitwarden(ctx context.Context) (bool, error) {
	itemRef := strings.TrimSpace(m.Config.BitwardenAgeKeyItem)
	if itemRef == "" {
		return false, nil
	}
	if _, err := m.Runner.LookPath("bw"); err != nil {
		return false, err
	}
	if err := m.ensureBitwardenSession(ctx); err != nil {
		return false, err
	}
	notes, err := m.bitwardenNotes(ctx, itemRef)
	if err != nil {
		return false, err
	}
	if !strings.Contains(notes, "AGE-SECRET-KEY-") {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(m.Config.AgeKeyFile), 0o700); err != nil {
		return false, err
	}
	if !strings.HasSuffix(notes, "\n") {
		notes += "\n"
	}
	if err := os.WriteFile(m.Config.AgeKeyFile, []byte(notes), 0o600); err != nil {
		return false, err
	}
	return true, os.Chmod(m.Config.AgeKeyFile, 0o600)
}

func (m Manager) ensureBitwardenSession(ctx context.Context) error {
	output, err := m.Runner.Output(ctx, "", "bw", "status", "--raw")
	if err != nil && len(output) == 0 {
		return err
	}
	status := parseBitwardenStatus(output)
	switch status {
	case "unlocked":
		return nil
	case "locked":
		session, err := m.interactiveOutput(ctx, "", "bw", "unlock", "--raw")
		if err != nil {
			return err
		}
		return setBWSession(session)
	case "unauthenticated":
		if err := m.Runner.Run(ctx, "", "bw", "login"); err != nil {
			return err
		}
		session, err := m.interactiveOutput(ctx, "", "bw", "unlock", "--raw")
		if err != nil {
			return err
		}
		return setBWSession(session)
	default:
		return fmt.Errorf("Bitwarden status is %q", status)
	}
}

func (m Manager) bitwardenNotes(ctx context.Context, itemRef string) (string, error) {
	output, err := m.Runner.Output(ctx, "", "bw", "get", "notes", itemRef)
	if err == nil && strings.TrimSpace(string(output)) != "" {
		return string(output), nil
	}
	itemOutput, itemErr := m.Runner.Output(ctx, "", "bw", "get", "item", itemRef)
	if itemErr != nil {
		if err != nil {
			return "", err
		}
		return "", itemErr
	}
	var item struct {
		Notes string `json:"notes"`
	}
	if err := json.Unmarshal(itemOutput, &item); err != nil {
		return "", err
	}
	return item.Notes, nil
}

func (m Manager) interactiveOutput(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	if runner, ok := m.Runner.(interactiveOutputRunner); ok {
		return runner.InteractiveOutput(ctx, dir, name, args...)
	}
	return m.Runner.Output(ctx, dir, name, args...)
}

func parseBitwardenStatus(output []byte) string {
	var status struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(output, &status); err == nil && strings.TrimSpace(status.Status) != "" {
		return strings.TrimSpace(status.Status)
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return "unknown"
	}
	return text
}

func setBWSession(output []byte) error {
	session := strings.TrimSpace(string(output))
	if session == "" {
		return fmt.Errorf("Bitwarden did not return a session token")
	}
	return os.Setenv("BW_SESSION", session)
}

func (m Manager) serversFileHasEntries() (bool, error) {
	return ssh.Store{Path: m.Config.ServersFile}.HasEntries()
}

func fileHasContent(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func isHelpArg(value string) bool {
	switch value {
	case "help", "--help", "-h", "-help":
		return true
	default:
		return false
	}
}

func showHelp() {
	ui.Title("Bootstrap")
	fmt.Printf("  %s dvv bootstrap [--force] [--skip-setup]\n\n", ui.Bold("Usage:"))
	helpSection("Actions")
	helpEntry("dvv bootstrap", "Restore AGE key, SSH list, and encrypted backup state")
	helpEntry("dvv bootstrap --force", "Overwrite the local SSH list from encrypted backup")
	helpEntry("dvv bootstrap --skip-setup", "Skip zsh completion and shortcut setup")
	fmt.Println()
	helpSection("Bitwarden")
	helpEntry("DVV_BW_AGE_KEY_ITEM", "Bitwarden item ID/name that stores the AGE private key in notes")
	helpEntry("DEVT_BW_AGE_KEY_ITEM", "Legacy equivalent still supported")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-34s %s\n", ui.Bold(command), description)
}
