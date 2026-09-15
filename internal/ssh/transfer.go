package ssh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/human"
	"github.com/VitorAllux/devtools/internal/terminal"
	"github.com/VitorAllux/devtools/internal/ui"
)

type transferDestination struct {
	Raw     string
	Label   string
	Path    string
	Custom  bool
	PerHost bool
}

type cleanupSummary struct {
	Files int
	Dirs  int
	Bytes int64
}

func (m *Manager) Download(ctx context.Context, entry Entry) error {
	if err := ValidateTarget(entry.Target); err != nil {
		return err
	}
	if !commandExists(m, "scp") {
		return fmt.Errorf("scp is required for SSH transfers")
	}
	remotePath, err := ui.Prompt("Remote file path, or directory path ending with /")
	if err != nil {
		return err
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return nil
	}
	destination, err := m.selectTransferDestination(ctx, entry)
	if err != nil || strings.TrimSpace(destination.Path) == "" {
		return err
	}
	recursive := false
	if strings.HasSuffix(remotePath, "/") {
		if !ui.Confirm("Remote path ends with /. Download this directory and all its contents?") {
			return nil
		}
		recursive = true
	}
	localPath := destination.Path
	if destination.PerHost {
		localPath = filepath.Join(localPath, transferEntryName(entry), time.Now().Format("2006-01-02"))
	}
	localPath = config.ExpandPath(localPath)
	if err := prepareTransferDestination(localPath, destination.PerHost); err != nil {
		return err
	}
	ui.Info("Download")
	ui.Info("  Host:   %s", entry.Name)
	ui.Info("  Remote: %s", remotePath)
	if destination.PerHost {
		ui.Info("  Local dir: %s", localPath)
	} else {
		ui.Info("  Local:  %s", localPath)
	}
	if !ui.Confirm("Start download?") {
		return nil
	}
	args := scpArgs(recursive, entry.Target+":"+remotePath, localPath)
	return m.runTransferInTerminal(ctx, "download", transferEntryName(entry), args)
}

func (m *Manager) Upload(ctx context.Context, entry Entry) error {
	if err := ValidateTarget(entry.Target); err != nil {
		return err
	}
	if !commandExists(m, "scp") {
		return fmt.Errorf("scp is required for SSH transfers")
	}
	localPath, err := m.selectLocalTransferPath(ctx)
	if err != nil || strings.TrimSpace(localPath) == "" {
		return err
	}
	localPath = config.ExpandPath(localPath)
	info, err := os.Lstat(localPath)
	if err != nil {
		return err
	}
	remotePath, err := ui.Prompt("Remote destination [~/]")
	if err != nil {
		return err
	}
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		remotePath = "~/"
	}
	recursive := info.IsDir()
	ui.Info("Upload")
	ui.Info("  Host:   %s", entry.Name)
	ui.Info("  Local:  %s", localPath)
	ui.Info("  Remote: %s", remotePath)
	if !ui.Confirm("Start upload?") {
		return nil
	}
	args := scpArgs(recursive, localPath, entry.Target+":"+remotePath)
	return m.runTransferInTerminal(ctx, "upload", transferEntryName(entry), args)
}

func (m *Manager) OpenDownloads(ctx context.Context) error {
	dir, err := m.ensureDownloadsDir()
	if err != nil {
		return err
	}
	return m.openDirectoryInTerminal(ctx, dir)
}

func (m *Manager) CleanDownloads(ctx context.Context) error {
	dir, err := m.ensureDownloadsDir()
	if err != nil {
		return err
	}
	if err := validateCleanDownloadsDir(m.Config, dir); err != nil {
		return err
	}
	summary := summarizeDir(dir)
	ui.Info("Clean SCP downloads")
	ui.Info("  Path:  %s", dir)
	ui.Info("  Files: %d", summary.Files)
	ui.Info("  Dirs:  %d", summary.Dirs)
	ui.Info("  Size:  %s", human.FormatBytes(summary.Bytes))
	if summary.Files == 0 && summary.Dirs == 0 {
		ui.Info("SCP downloads are already clean")
		return nil
	}
	if !ui.Confirm("Remove all files inside SCP downloads?") {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	ui.OK("Deleted SCP downloads content: %s", human.FormatBytes(summary.Bytes))
	return nil
}

func (m *Manager) ensureDownloadsDir() (string, error) {
	dir := config.ExpandPath(strings.TrimSpace(m.Config.Project.SSH.Transfer.DownloadsDir))
	if dir == "" {
		dir = config.ExpandPath(config.DefaultProjectConfig().SSH.Transfer.DownloadsDir)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func (m *Manager) selectTransferDestination(ctx context.Context, entry Entry) (transferDestination, error) {
	downloadsDir, err := m.ensureDownloadsDir()
	if err != nil {
		return transferDestination{}, err
	}
	cwd, _ := os.Getwd()
	options := []transferDestination{
		{Raw: "default", Label: "Default downloads dir", Path: downloadsDir, PerHost: true},
	}
	if cwd != "" {
		options = append(options, transferDestination{Raw: "current", Label: "Current directory", Path: cwd})
	}
	if workspaceRoot := strings.TrimSpace(m.Config.Project.Workspace.Root); workspaceRoot != "" {
		options = append(options, transferDestination{Raw: "workspace", Label: "Workspace root", Path: config.ExpandPath(workspaceRoot)})
	}
	options = append(options, transferDestination{Raw: "custom", Label: "Custom path", Custom: true})

	selected := options[0]
	if _, err := m.Runner.LookPath("fzf"); err == nil {
		var builder strings.Builder
		for _, option := range options {
			builder.WriteString(ui.FZFHiddenRow(option.Raw, transferDestinationRow(option)))
			builder.WriteByte('\n')
		}
		args := ui.FZFHub{
			Prompt:      ui.Crown("destination") + ui.Muted("> "),
			BorderLabel: "dvv ssh download",
			Shortcuts: []ui.FZFShortcut{
				{Label: "Enter", Description: "select destination"},
				{Label: "Esc", Description: "cancel"},
			},
			ExtraArgs: ui.FZFHiddenRowArgs(),
		}.Args()
		output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
		if err != nil && len(output) == 0 {
			return transferDestination{}, nil
		}
		raw := ui.FZFSelectedRaw(strings.TrimSpace(string(output)))
		found := false
		for _, option := range options {
			if option.Raw == raw {
				selected = option
				found = true
				break
			}
		}
		if !found {
			return transferDestination{}, nil
		}
	} else {
		for index, option := range options {
			fmt.Printf("  %2d. %-24s %s\n", index+1, option.Label, option.Path)
		}
		value, err := ui.Prompt("Destination")
		if err != nil {
			return transferDestination{}, err
		}
		if strings.TrimSpace(value) == "" {
			return selected, nil
		}
		index, ok := parseEntryIndex(value, len(options))
		if !ok {
			return transferDestination{}, fmt.Errorf("invalid destination selection: %s", value)
		}
		selected = options[index]
	}
	if selected.Custom {
		value, err := ui.Prompt("Local destination")
		if err != nil {
			return transferDestination{}, err
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return transferDestination{}, nil
		}
		selected.Path = config.ExpandPath(value)
	}
	_ = entry
	return selected, nil
}

func (m *Manager) selectLocalTransferPath(ctx context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		cwd = "."
	}
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		return ui.Prompt("Local path")
	}
	candidates := localTransferCandidates(cwd)
	if len(candidates) == 0 {
		return ui.Prompt("Local path")
	}
	var builder strings.Builder
	for _, candidate := range candidates {
		builder.WriteString(ui.FZFHiddenRow(candidate, localTransferRow(candidate)))
		builder.WriteByte('\n')
	}
	args := ui.FZFHub{
		Prompt:       ui.Crown("local") + ui.Muted("> "),
		BorderLabel:  "dvv ssh upload",
		Preview:      localPathPreviewCommand(),
		PreviewLabel: "local path",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "select local path"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: ui.FZFHiddenRowArgs(),
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
	if err != nil && len(output) == 0 {
		return "", nil
	}
	return ui.FZFSelectedRaw(strings.TrimSpace(string(output))), nil
}

func (m *Manager) runTransferInTerminal(ctx context.Context, action string, name string, scpArgs []string) error {
	if !commandExists(m, "tmux") {
		return fmt.Errorf("tmux is required for SSH transfers")
	}
	session := transferSessionName(action, name)
	command := transferShellCommand(scpArgs)
	if err := m.Runner.Run(ctx, "", "tmux", "new-session", "-ds", session, "-n", action, command); err != nil {
		return err
	}
	if err := (terminal.Launcher{Runner: m.Runner, Preferred: m.terminalLauncherPreference()}).Open(ctx, "tmux", "attach", "-t", session); err != nil {
		return fmt.Errorf("%w; attach manually with: tmux attach -t %s", err, session)
	}
	return nil
}

func (m *Manager) openDirectoryInTerminal(ctx context.Context, dir string) error {
	if !commandExists(m, "tmux") {
		return fmt.Errorf("tmux is required to open SCP downloads")
	}
	if os.Getenv("TMUX") != "" {
		return m.Runner.Run(ctx, "", "tmux", "new-window", "-c", dir, "-n", "scp-downloads")
	}
	session := "dvv-scp-downloads-" + time.Now().Format("150405000")
	if err := m.Runner.Run(ctx, "", "tmux", "new-session", "-ds", session, "-c", dir, "-n", "downloads"); err != nil {
		return err
	}
	if err := (terminal.Launcher{Runner: m.Runner, Preferred: m.terminalLauncherPreference()}).Open(ctx, "tmux", "attach", "-t", session); err != nil {
		return fmt.Errorf("%w; attach manually with: tmux attach -t %s", err, session)
	}
	return nil
}

func scpArgs(recursive bool, source string, destination string) []string {
	args := []string{}
	if recursive {
		args = append(args, "-r")
	}
	return append(args, source, destination)
}

func transferShellCommand(args []string) string {
	parts := []string{"scp"}
	parts = append(parts, args...)
	for index, part := range parts {
		parts[index] = shellQuote(part)
	}
	command := strings.Join(parts, " ")
	return command + `; code=$?; printf '\nSCP finished with exit code %s. Press Enter to close.' "$code"; read _; exit "$code"`
}

func prepareTransferDestination(path string, directory bool) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("local destination cannot be empty")
	}
	if directory {
		return os.MkdirAll(path, 0o700)
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil
	}
	if strings.HasSuffix(path, string(filepath.Separator)) {
		return os.MkdirAll(path, 0o700)
	}
	return os.MkdirAll(filepath.Dir(path), 0o700)
}

func validateCleanDownloadsDir(cfg *config.Config, dir string) error {
	abs, err := filepath.Abs(config.ExpandPath(dir))
	if err != nil {
		return err
	}
	if abs == string(filepath.Separator) || abs == "." {
		return fmt.Errorf("refusing to clean unsafe SCP downloads path: %s", dir)
	}
	if home, err := os.UserHomeDir(); err == nil {
		if samePath(abs, home) {
			return fmt.Errorf("refusing to clean home directory as SCP downloads path")
		}
	}
	if cfg != nil {
		for _, unsafe := range []string{cfg.RootDir, cfg.ConfigDir} {
			if strings.TrimSpace(unsafe) != "" && samePath(abs, config.ExpandPath(unsafe)) {
				return fmt.Errorf("refusing to clean unsafe SCP downloads path: %s", abs)
			}
		}
	}
	return nil
}

func samePath(a string, b string) bool {
	absA, errA := filepath.Abs(config.ExpandPath(a))
	absB, errB := filepath.Abs(config.ExpandPath(b))
	return errA == nil && errB == nil && absA == absB
}

func summarizeDir(root string) cleanupSummary {
	summary := cleanupSummary{}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			summary.Files++
			return nil
		}
		if entry.IsDir() {
			summary.Dirs++
			return nil
		}
		summary.Files++
		summary.Bytes += info.Size()
		return nil
	})
	return summary
}

func localTransferCandidates(root string) []string {
	const maxEntries = 500
	const maxDepth = 4
	root, _ = filepath.Abs(root)
	candidates := []string{root}
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}
		if len(candidates) >= maxEntries {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if entry.IsDir() && shouldSkipTransferCandidateDir(name) {
			return filepath.SkipDir
		}
		if depth(root, path) > maxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		candidates = append(candidates, path)
		return nil
	})
	sort.Strings(candidates)
	return candidates
}

func shouldSkipTransferCandidateDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "dist", "build", ".cache", ".next":
		return true
	default:
		return false
	}
}

func depth(root string, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return len(strings.Split(rel, string(filepath.Separator)))
}

func transferDestinationRow(option transferDestination) string {
	path := option.Path
	if option.Custom {
		path = "ask"
	}
	return fmt.Sprintf("%s  %s", ui.Accent(fixedWidth(option.Label, 24)), ui.Muted(path))
}

func localTransferRow(path string) string {
	info, err := os.Lstat(path)
	kind := "file"
	size := "-"
	if err == nil {
		if info.IsDir() {
			kind = "dir"
		} else if info.Mode()&os.ModeSymlink != 0 {
			kind = "link"
		} else {
			size = human.FormatBytes(info.Size())
		}
	}
	return fmt.Sprintf("%s  %s  %s", ui.Gold(fixedWidth(kind, 5)), ui.Muted(fixedWidth(size, 10)), ui.Accent(path))
}

func localPathPreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
printf "%sLocal path%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-7s%s %s\n" "$dvv_label" "Path" "$dvv_reset" "$raw"
if [ -d "$raw" ]; then
  printf "  %s%-7s%s directory\n" "$dvv_label" "Type" "$dvv_reset"
elif [ -f "$raw" ]; then
  printf "  %s%-7s%s file\n" "$dvv_label" "Type" "$dvv_reset"
else
  printf "  %s%-7s%s other\n" "$dvv_label" "Type" "$dvv_reset"
fi
' sh {}`
}

func transferEntryName(entry Entry) string {
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = entry.Target
	}
	name = tmuxSessionUnsafeChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-.")
	if name == "" {
		return "target"
	}
	return name
}

func transferSessionName(action string, name string) string {
	name = tmuxSessionUnsafeChars.ReplaceAllString(strings.TrimSpace(name), "-")
	name = strings.Trim(name, "-.")
	if name == "" {
		name = "target"
	}
	return "dvv-scp-" + action + "-" + name + "-" + time.Now().Format("150405000")
}
