package db

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"unicode"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/human"
	"github.com/VitorAllux/devtools/internal/run"
	"github.com/VitorAllux/devtools/internal/ui"
)

type Manager struct {
	Config       *config.Config
	Runner       run.Runner
	defaultsFile string
}

type Action struct {
	Name        string
	Label       string
	Description string
}

type DatabaseInfo struct {
	Name      string
	SizeBytes int64
	SizeKnown bool
}

type DumpFile struct {
	Name      string
	SizeBytes int64
	SizeKnown bool
}

type DriveDumpEntry struct {
	ID       string
	Name     string
	Path     string
	Size     int64
	MimeType string
	ModTime  string
	IsDir    bool
}

type driveDumpSelectionRow struct {
	Raw     string
	Kind    string
	Name    string
	Size    string
	ModTime string
	Display string
}

type driveDumpBrowserSelection struct {
	Key  string
	Raws []string
}

type driveFolderRef struct {
	ID          string
	ResourceKey string
}

const (
	browseDriveDumpsOption = "[+] Browse Google Drive dumps"
	pasteDriveLinkOption   = "[+] Paste Google Drive link or ID"
	driveDumpNameWidth     = 56
	driveDumpMoveKey       = "M"
	driveDumpRefreshKey    = "R"
)

var sqlTablePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^--\s*(?:table structure for table|dumping data for table)\s+(.+)`),
	regexp.MustCompile(`(?i)^CREATE\s+(?:TEMPORARY\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(.+)`),
	regexp.MustCompile(`(?i)^INSERT\s+(?:IGNORE\s+)?INTO\s+(.+)`),
	regexp.MustCompile(`(?i)^REPLACE\s+(?:LOW_PRIORITY\s+|DELAYED\s+)?INTO\s+(.+)`),
	regexp.MustCompile(`(?i)^ALTER\s+TABLE\s+(.+)`),
	regexp.MustCompile(`(?i)^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(.+)`),
	regexp.MustCompile(`(?i)^LOCK\s+TABLES\s+(.+)`),
}

func Run(ctx context.Context, cfg *config.Config, runner run.Runner, args []string) error {
	manager := &Manager{Config: cfg, Runner: runner}
	defer manager.cleanup()

	if len(args) == 0 {
		return manager.Hub(ctx)
	}
	switch args[0] {
	case "help", "--help", "-h":
		showHelp()
		return nil
	case "create":
		return manager.CommandCreate(ctx, args[1:])
	case "drop":
		return manager.CommandDrop(ctx)
	case "truncate":
		return manager.CommandTruncate(ctx)
	case "clean":
		return manager.CommandClean(ctx)
	case "import":
		return manager.CommandImport(ctx)
	default:
		return fmt.Errorf("unknown db action: %s", args[0])
	}
}

func (m *Manager) Hub(ctx context.Context) error {
	for {
		action, err := m.selectAction(ctx)
		if err != nil || action == "" {
			return err
		}
		switch action {
		case "create":
			err = m.CommandCreate(ctx, nil)
		case "drop":
			err = m.CommandDrop(ctx)
		case "truncate":
			err = m.CommandTruncate(ctx)
		case "clean":
			err = m.CommandClean(ctx)
		case "import":
			err = m.CommandImport(ctx)
		}
		if err != nil {
			ui.Error("%v", err)
		}
	}
}

func (m *Manager) CommandCreate(ctx context.Context, args []string) error {
	name := strings.TrimSpace(strings.Join(args, " "))
	if name == "" {
		var err error
		name, err = ui.Prompt("Database name")
		if err != nil {
			return err
		}
	}
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	if err := m.prepareAuth(); err != nil {
		return err
	}
	return m.ensureDatabase(ctx, name, true)
}

func (m *Manager) CommandDrop(ctx context.Context) error {
	selected, err := m.selectDatabases(ctx, true, "Select databases to DROP")
	if err != nil || len(selected) == 0 {
		return err
	}
	for _, name := range selected {
		ui.Warn("Drop database: %s", name)
	}
	if !ui.Confirm(fmt.Sprintf("Drop %d database(s)?", len(selected))) {
		return nil
	}
	for _, name := range selected {
		if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "dropping", Subject: "database", Detail: name}, func() error {
			return m.mysql(ctx, "-e", "DROP DATABASE "+identifier(name)+";")
		}); err != nil {
			return err
		}
		ui.OK("Dropped database %s", name)
	}
	return nil
}

func (m *Manager) CommandTruncate(ctx context.Context) error {
	selected, err := m.selectDatabases(ctx, true, "Select databases to TRUNCATE")
	if err != nil || len(selected) == 0 {
		return err
	}
	for _, name := range selected {
		ui.Warn("Truncate database: %s", name)
	}
	if !ui.Confirm(fmt.Sprintf("Truncate %d database(s)?", len(selected))) {
		return nil
	}
	for _, name := range selected {
		if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "truncating", Subject: "database", Detail: name}, func() error {
			return m.truncate(ctx, name)
		}); err != nil {
			return err
		}
		ui.OK("Truncated database %s", name)
	}
	return nil
}

func (m *Manager) CommandClean(ctx context.Context) error {
	dumps, err := m.dumpFileInfos()
	if err != nil {
		return err
	}
	if len(dumps) == 0 {
		ui.Info("No dumps found in %s", m.Config.Project.DB.DumpsDir)
		return nil
	}
	selected, err := m.selectDumpFiles(ctx, "Delete Dumps", dumps, true)
	if err != nil || len(selected) == 0 {
		return err
	}
	for _, file := range selected {
		ui.Warn("Delete dump: %s", file)
	}
	if !ui.Confirm(fmt.Sprintf("Delete %d dump file(s)?", len(selected))) {
		return nil
	}
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "deleting", Subject: fmt.Sprintf("%d dump file(s)", len(selected))}, func() error {
		for _, file := range selected {
			if err := os.Remove(filepath.Join(m.Config.Project.DB.DumpsDir, file)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	ui.OK("Deleted %d dump file(s)", len(selected))
	return nil
}

func (m *Manager) CommandImport(ctx context.Context) error {
	if err := os.MkdirAll(m.Config.Project.DB.DumpsDir, 0o755); err != nil {
		return err
	}
	dump, err := m.selectOrDownloadDump(ctx)
	if err != nil || dump == "" {
		return err
	}
	dbName, err := m.selectImportDatabase(ctx)
	if err != nil {
		return err
	}
	ui.Info("Local file: %s", dump)
	ui.Info("Target DB:  %s", dbName)
	if !ui.Confirm("Start import?") {
		return nil
	}
	if err := m.ensureDatabase(ctx, dbName, false); err != nil {
		return err
	}
	return m.importDump(ctx, dump, dbName)
}

func (m *Manager) selectAction(ctx context.Context) (string, error) {
	actions := []Action{
		{Name: "create", Label: "Create database", Description: "Create a new local database"},
		{Name: "import", Label: "Import dump", Description: "Import a local or Google Drive SQL dump"},
		{Name: "truncate", Label: "Truncate databases", Description: "Delete all table data from selected databases"},
		{Name: "drop", Label: "Drop databases", Description: "Delete selected databases completely"},
		{Name: "clean", Label: "Clean dumps", Description: "Delete selected local dump files"},
	}
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		for index, action := range actions {
			fmt.Printf("  %2d. %-20s %s\n", index+1, action.Label, ui.Dim(action.Description))
		}
		value, err := ui.Prompt("DB action")
		if err != nil {
			return "", err
		}
		index, ok := parseIndex(value, len(actions))
		if !ok {
			return "", fmt.Errorf("invalid DB action: %s", value)
		}
		return actions[index].Name, nil
	}
	var builder strings.Builder
	builder.WriteString(dbActionLine("__dvv_header__", "", "", dbActionHeader()))
	builder.WriteByte('\n')
	for index, action := range actions {
		builder.WriteString(dbActionLine(action.Name, action.Label, action.Description, dbActionRow(index, action)))
		builder.WriteByte('\n')
	}
	args := ui.FZFHub{
		Prompt:        ui.Crown("db") + ui.Muted("> "),
		BorderLabel:   "dvv db",
		Preview:       dbActionPreviewCommand(),
		PreviewLabel:  "action panel",
		PreviewWindow: "right,38%,border-rounded,wrap",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "run action"},
			{Label: "Esc", Description: "exit"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=4",
			"--nth=1,2,3,4",
			"--header-lines=1",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
	if err != nil && len(output) == 0 {
		return "", nil
	}
	return ui.FZFSelectedRaw(strings.TrimSpace(string(output))), nil
}

func (m *Manager) selectDatabases(ctx context.Context, multi bool, label string) ([]string, error) {
	if err := m.prepareAuth(); err != nil {
		return nil, err
	}
	var databases []DatabaseInfo
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "fetching", Subject: "databases"}, func() error {
		var err error
		databases, err = m.databaseInfos(ctx)
		return err
	}); err != nil {
		return nil, err
	}
	if len(databases) == 0 {
		return nil, fmt.Errorf("no user databases found")
	}
	return m.selectDatabaseInfos(ctx, label, databases, multi)
}

func (m *Manager) selectDatabaseInfos(ctx context.Context, label string, databases []DatabaseInfo, multi bool) ([]string, error) {
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		for index, database := range databases {
			fmt.Printf("  %2d. %-32s %s\n", index+1, database.Name, databaseSize(database))
		}
		selected, err := ui.Prompt(label)
		if err != nil {
			return nil, err
		}
		values := databaseNames(databases)
		if multi {
			return valuesByIndexes(values, selected), nil
		}
		index, ok := parseIndex(selected, len(databases))
		if !ok {
			return nil, fmt.Errorf("invalid selection: %s", selected)
		}
		return []string{databases[index].Name}, nil
	}
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(databaseInfoHeader()))
	builder.WriteByte('\n')
	for index, database := range databases {
		builder.WriteString(ui.FZFHiddenRow(database.Name, databaseInfoRow(index, database)))
		builder.WriteByte('\n')
	}
	fzf := ui.FZFHub{
		Prompt:      ui.Crown("database") + ui.Muted("> "),
		BorderLabel: label,
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "confirm"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(), "--header-lines=1"),
	}
	if multi {
		fzf.Shortcuts = append([]ui.FZFShortcut{{Label: "Tab", Description: "mark"}}, fzf.Shortcuts...)
		fzf.ExtraArgs = append(fzf.ExtraArgs, "--multi")
	}
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", fzf.Args()...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}
	return nonEmptyRawLines(string(output)), nil
}

func (m *Manager) selectValues(ctx context.Context, label string, values []string, multi bool) ([]string, error) {
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		for index, value := range values {
			fmt.Printf("  %2d. %s\n", index+1, value)
		}
		selected, err := ui.Prompt(label)
		if err != nil {
			return nil, err
		}
		if multi {
			return valuesByIndexes(values, selected), nil
		}
		index, ok := parseIndex(selected, len(values))
		if !ok {
			return nil, fmt.Errorf("invalid selection: %s", selected)
		}
		return []string{values[index]}, nil
	}
	var builder strings.Builder
	for _, value := range values {
		builder.WriteString(ui.FZFHiddenRow(value, ui.Accent(value)))
		builder.WriteByte('\n')
	}
	fzf := ui.FZFHub{
		Prompt:      ui.Crown("select") + ui.Muted("> "),
		BorderLabel: label,
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "confirm"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: ui.FZFHiddenRowArgs(),
	}
	if multi {
		fzf.Shortcuts = append([]ui.FZFShortcut{{Label: "Tab", Description: "mark"}}, fzf.Shortcuts...)
		fzf.ExtraArgs = append(fzf.ExtraArgs, "--multi")
	}
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", fzf.Args()...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}
	return nonEmptyRawLines(string(output)), nil
}

func (m *Manager) selectOrDownloadDump(ctx context.Context) (string, error) {
	dumps, err := m.dumpFileInfos()
	if err != nil {
		return "", err
	}
	options := append([]DumpFile{{Name: browseDriveDumpsOption}, {Name: pasteDriveLinkOption}}, dumps...)
	selected, err := m.selectDumpFiles(ctx, "Select dump", options, false)
	if err != nil || len(selected) == 0 {
		return "", err
	}
	switch selected[0] {
	case browseDriveDumpsOption:
		return m.downloadDumpFromDriveBrowser(ctx)
	case pasteDriveLinkOption:
		return m.downloadDump(ctx)
	default:
		return filepath.Join(m.Config.Project.DB.DumpsDir, selected[0]), nil
	}
}

func (m *Manager) selectDumpFiles(ctx context.Context, label string, dumps []DumpFile, multi bool) ([]string, error) {
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		for index, dump := range dumps {
			fmt.Printf("  %2d. %-42s %s\n", index+1, dump.Name, dumpFileSize(dump))
		}
		selected, err := ui.Prompt(label)
		if err != nil {
			return nil, err
		}
		values := dumpFileNames(dumps)
		if multi {
			return valuesByIndexes(values, selected), nil
		}
		index, ok := parseIndex(selected, len(dumps))
		if !ok {
			return nil, fmt.Errorf("invalid selection: %s", selected)
		}
		return []string{dumps[index].Name}, nil
	}
	var builder strings.Builder
	builder.WriteString(ui.FZFHiddenHeader(dumpFileHeader()))
	builder.WriteByte('\n')
	for index, dump := range dumps {
		builder.WriteString(ui.FZFHiddenRow(dump.Name, dumpFileRow(index, dump)))
		builder.WriteByte('\n')
	}
	fzf := ui.FZFHub{
		Prompt:      ui.Crown("dump") + ui.Muted("> "),
		BorderLabel: label,
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "confirm"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: append(ui.FZFHiddenRowArgs(), "--header-lines=1"),
	}
	if multi {
		fzf.Shortcuts = append([]ui.FZFShortcut{{Label: "Tab", Description: "mark"}}, fzf.Shortcuts...)
		fzf.ExtraArgs = append(fzf.ExtraArgs, "--multi")
	}
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", fzf.Args()...)
	if err != nil && len(output) == 0 {
		return nil, nil
	}
	return nonEmptyRawLines(string(output)), nil
}

func dumpFileNames(dumps []DumpFile) []string {
	names := make([]string, 0, len(dumps))
	for _, dump := range dumps {
		names = append(names, dump.Name)
	}
	return names
}

func (m *Manager) downloadDump(ctx context.Context) (string, error) {
	remote, remotes, err := m.promptRcloneRemote(ctx)
	if err != nil {
		return "", err
	}
	if value, err := ui.Prompt("Rclone remote [" + remote + "]"); err != nil {
		return "", err
	} else if strings.TrimSpace(value) != "" {
		remote = normalizeRcloneRemote(value)
	}
	if err := validateRcloneRemote(remote, remotes); err != nil {
		return "", err
	}
	fileID, err := ui.Prompt("Google Drive File ID or link")
	if err != nil {
		return "", err
	}
	fileID, ok := extractDriveFileID(fileID)
	if !ok {
		return "", fmt.Errorf("invalid Google Drive File ID/link")
	}
	fileName, err := ui.Prompt("Local file name")
	if err != nil {
		return "", err
	}
	fileName = normalizeDownloadDumpFileName(fileName, fileID)
	return m.downloadDriveFileByID(ctx, remote, fileID, fileName)
}

func (m *Manager) downloadDumpFromDriveBrowser(ctx context.Context) (string, error) {
	remote, remotes, err := m.promptRcloneRemote(ctx)
	if err != nil {
		return "", err
	}
	if err := validateRcloneRemote(remote, remotes); err != nil {
		return "", err
	}
	folder, ok := extractDriveFolderRef(m.Config.Project.DB.DriveFolderID)
	if !ok {
		return "", fmt.Errorf("DVV_DB_DRIVE_FOLDER_ID is not configured; set it with `dvv config` or use %s", pasteDriveLinkOption)
	}
	entry, ok, err := m.browseDriveDump(ctx, remote, folder)
	if err != nil || !ok {
		return "", err
	}
	if strings.TrimSpace(entry.ID) == "" {
		return "", fmt.Errorf("selected Drive file has no ID: %s", entry.Name)
	}
	return m.downloadDriveFileByID(ctx, remote, entry.ID, entry.Name)
}

func (m *Manager) promptRcloneRemote(ctx context.Context) (string, []string, error) {
	if _, err := m.Runner.LookPath("rclone"); err != nil {
		return "", nil, fmt.Errorf("rclone is required to download dumps")
	}
	remotes, err := m.rcloneRemotes(ctx)
	if err != nil {
		return normalizeRcloneRemote(m.Config.Project.DB.RcloneRemote), nil, nil
	}
	remote := normalizeRcloneRemote(m.Config.Project.DB.RcloneRemote)
	return remote, remotes, nil
}

func validateRcloneRemote(remote string, remotes []string) error {
	if remote == "" {
		return fmt.Errorf("rclone remote cannot be empty")
	}
	if remotes == nil {
		return nil
	}
	if len(remotes) == 0 {
		return fmt.Errorf("rclone has no configured remotes; run `rclone config` before downloading dumps")
	}
	if !rcloneRemoteExists(remotes, remote) {
		return unknownRcloneRemoteError(remote, remotes)
	}
	return nil
}

func (m *Manager) downloadDriveFileByID(ctx context.Context, remote string, fileID string, fileName string) (string, error) {
	fileName = normalizeDownloadDumpFileName(fileName, fileID)
	dest := filepath.Join(m.Config.Project.DB.DumpsDir, filepath.Base(fileName))
	if err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "downloading", Subject: filepath.Base(dest), ShowResult: true, SuccessAction: "downloaded"}, func() error {
		return run.Quiet(ctx, m.Runner, "", "rclone", "backend", "copyid", remote+":", fileID, dest)
	}); err != nil {
		return "", err
	}
	return dest, nil
}

func (m *Manager) browseDriveDump(ctx context.Context, remote string, rootFolder driveFolderRef) (DriveDumpEntry, bool, error) {
	currentPath := ""
	for {
		var entries []DriveDumpEntry
		folder := dbDefaultString(currentPath, "/")
		err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "listing", Subject: "Drive dumps " + folder}, func() error {
			var err error
			entries, err = m.driveDumpEntries(ctx, remote, rootFolder, currentPath)
			return err
		})
		if err != nil {
			return DriveDumpEntry{}, false, err
		}
		selection, ok, err := m.selectDriveDumpEntry(ctx, currentPath, entries)
		if err != nil || !ok {
			return DriveDumpEntry{}, false, err
		}
		if selection.Key == driveDumpRefreshKey {
			continue
		}
		if selection.Key == driveDumpMoveKey {
			if err := m.moveSelectedDriveDumps(ctx, remote, rootFolder, currentPath, entries, selection.Raws); err != nil {
				return DriveDumpEntry{}, false, err
			}
			continue
		}
		if len(selection.Raws) == 0 {
			continue
		}
		raw := selection.Raws[0]
		if raw == "__parent__" {
			currentPath = pathpkg.Dir(strings.TrimSuffix(currentPath, "/"))
			if currentPath == "." {
				currentPath = ""
			}
			continue
		}
		index, ok := parseDriveSelection(raw, len(entries))
		if !ok {
			return DriveDumpEntry{}, false, fmt.Errorf("invalid Drive selection: %s", raw)
		}
		entry := entries[index]
		if entry.IsDir {
			currentPath = drivePathJoin(currentPath, entry.Name)
			continue
		}
		return entry, true, nil
	}
}

func (m *Manager) moveSelectedDriveDumps(ctx context.Context, remote string, rootFolder driveFolderRef, currentPath string, entries []DriveDumpEntry, rawSelections []string) error {
	selected, err := driveDumpFilesFromSelections(rawSelections, entries)
	if err != nil {
		ui.Warn("%s", err.Error())
		return nil
	}
	folders := driveDumpFolders(entries)
	if len(folders) == 0 {
		ui.Warn("No destination folders in %s", dbDefaultString(currentPath, "/"))
		return nil
	}
	destination, ok, err := m.selectDriveMoveDestination(ctx, currentPath, folders)
	if err != nil || !ok {
		return err
	}
	if !ui.Confirm(fmt.Sprintf("Move %d file(s) to %s?", len(selected), drivePathJoin(dbDefaultString(currentPath, "/"), destination.Name))) {
		return nil
	}
	for _, entry := range selected {
		if err := m.moveDriveDump(ctx, remote, rootFolder, currentPath, entry, destination); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) moveDriveDump(ctx context.Context, remote string, rootFolder driveFolderRef, currentPath string, entry DriveDumpEntry, destination DriveDumpEntry) error {
	sourcePath := drivePathJoin(currentPath, entry.Name)
	destPath := drivePathJoin(drivePathJoin(currentPath, destination.Name), entry.Name)
	subject := entry.Name + " -> " + destination.Name
	return ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "moving", Subject: subject, ShowResult: true, SuccessAction: "moved"}, func() error {
		return run.Quiet(ctx, m.Runner, "", "rclone", driveRootArgs([]string{"moveto", driveRemotePath(remote, sourcePath), driveRemotePath(remote, destPath)}, rootFolder)...)
	})
}

func (m *Manager) driveDumpEntries(ctx context.Context, remote string, rootFolder driveFolderRef, currentPath string) ([]DriveDumpEntry, error) {
	remotePath := driveRemotePath(remote, currentPath)
	out, err := m.driveDumpEntriesOutput(ctx, remotePath, rootFolder)
	if err != nil {
		return nil, err
	}
	return parseDriveDumpEntries(out)
}

func (m *Manager) driveDumpEntriesOutput(ctx context.Context, remotePath string, rootFolder driveFolderRef) ([]byte, error) {
	args := driveRootArgs([]string{"lsjson", remotePath}, rootFolder)
	return m.Runner.Output(ctx, "", "rclone", args...)
}

func driveRootArgs(args []string, rootFolder driveFolderRef) []string {
	args = append(args, "--drive-root-folder-id", rootFolder.ID)
	if strings.TrimSpace(rootFolder.ResourceKey) != "" {
		args = append(args, "--drive-resource-key", rootFolder.ResourceKey)
	}
	return args
}

func parseDriveDumpEntries(out []byte) ([]DriveDumpEntry, error) {
	var entries []DriveDumpEntry
	jsonStart := bytes.IndexByte(out, '[')
	if jsonStart < 0 {
		return nil, fmt.Errorf("rclone lsjson returned no JSON output")
	}
	var lastErr error
	for jsonStart >= 0 {
		decoder := json.NewDecoder(bytes.NewReader(out[jsonStart:]))
		if err := decoder.Decode(&entries); err == nil {
			break
		} else {
			lastErr = err
		}
		next := bytes.IndexByte(out[jsonStart+1:], '[')
		if next < 0 {
			return nil, lastErr
		}
		jsonStart += next + 1
	}
	filtered := make([]DriveDumpEntry, 0, len(entries))
	for _, entry := range entries {
		entry.Name = strings.TrimSpace(entry.Name)
		if entry.Name == "" {
			continue
		}
		if entry.IsDir || isDriveDumpFileName(entry.Name) {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].IsDir != filtered[j].IsDir {
			return filtered[i].IsDir
		}
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})
	return filtered, nil
}

func driveRemotePath(remote string, currentPath string) string {
	remotePath := normalizeRcloneRemote(remote) + ":"
	currentPath = strings.Trim(strings.TrimSpace(currentPath), "/")
	if currentPath != "" {
		remotePath += currentPath
	}
	return remotePath
}

func driveDumpFilesFromSelections(rawSelections []string, entries []DriveDumpEntry) ([]DriveDumpEntry, error) {
	selected := make([]DriveDumpEntry, 0, len(rawSelections))
	seen := map[int]bool{}
	for _, raw := range rawSelections {
		index, ok := parseDriveSelection(raw, len(entries))
		if !ok || seen[index] {
			continue
		}
		seen[index] = true
		entry := entries[index]
		if entry.IsDir {
			continue
		}
		selected = append(selected, entry)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("mark one or more files with Tab before moving")
	}
	return selected, nil
}

func driveDumpFolders(entries []DriveDumpEntry) []DriveDumpEntry {
	folders := make([]DriveDumpEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			folders = append(folders, entry)
		}
	}
	return folders
}

func driveDumpRawFromLine(line string) string {
	raw, _, _ := strings.Cut(strings.TrimSpace(line), "\t")
	return strings.TrimSpace(raw)
}

func (m *Manager) selectDriveDumpEntry(ctx context.Context, currentPath string, entries []DriveDumpEntry) (driveDumpBrowserSelection, bool, error) {
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		rows := driveDumpSelectionRows(currentPath, entries)
		for index, row := range rows {
			fmt.Printf("  %2d. %s\n", index+1, row.Display)
		}
		value, err := ui.Prompt("Drive dump")
		if err != nil {
			return driveDumpBrowserSelection{}, false, err
		}
		index, ok := parseIndex(value, len(rows))
		if !ok {
			return driveDumpBrowserSelection{}, false, fmt.Errorf("invalid Drive selection: %s", value)
		}
		return driveDumpBrowserSelection{Raws: []string{rows[index].Raw}}, true, nil
	}
	rows := driveDumpSelectionRows(currentPath, entries)
	var builder strings.Builder
	builder.WriteString(driveDumpFZFLine("__dvv_header__", "", "", "", "", driveDumpHeader()))
	builder.WriteByte('\n')
	for _, row := range rows {
		builder.WriteString(driveDumpFZFLine(row.Raw, row.Kind, row.Name, row.Size, row.ModTime, row.Display))
		builder.WriteByte('\n')
	}
	args := ui.FZFHub{
		Prompt:       ui.Crown("drive") + ui.Muted("> "),
		BorderLabel:  "Google Drive dumps",
		Preview:      driveDumpPreviewCommand(currentPath),
		PreviewLabel: "drive item",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Tab", Description: "mark files"},
			{Key: driveDumpMoveKey, Label: "Shift+M", Description: "move marked files"},
			{Key: driveDumpRefreshKey, Label: "Shift+R", Description: "refresh folder"},
			{Label: "Enter", Description: "open/import"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=6..",
			"--nth=2,3,6..",
			"--header-lines=1",
			"--multi",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
	if err != nil && len(output) == 0 {
		return driveDumpBrowserSelection{}, false, nil
	}
	if key := strings.TrimSpace(string(output)); key == driveDumpRefreshKey {
		return driveDumpBrowserSelection{Key: key}, true, nil
	}
	key, selected := ui.ParseFZFExpectOutput(string(output))
	raws := make([]string, 0, len(selected))
	for _, selection := range selected {
		raw := driveDumpRawFromLine(selection)
		if raw != "" && raw != "__dvv_header__" {
			raws = append(raws, raw)
		}
	}
	return driveDumpBrowserSelection{Key: key, Raws: raws}, true, nil
}

func (m *Manager) selectDriveMoveDestination(ctx context.Context, currentPath string, folders []DriveDumpEntry) (DriveDumpEntry, bool, error) {
	if _, err := m.Runner.LookPath("fzf"); err != nil {
		for index, folder := range folders {
			fmt.Printf("  %2d. %s\n", index+1, folder.Name)
		}
		value, err := ui.Prompt("Move to folder")
		if err != nil {
			return DriveDumpEntry{}, false, err
		}
		index, ok := parseIndex(value, len(folders))
		if !ok {
			return DriveDumpEntry{}, false, fmt.Errorf("invalid folder selection: %s", value)
		}
		return folders[index], true, nil
	}
	var builder strings.Builder
	builder.WriteString(driveDumpFZFLine("__dvv_header__", "", "", "", "", driveDumpHeader()))
	builder.WriteByte('\n')
	for index, folder := range folders {
		row := driveDumpSelectionRow{Raw: fmt.Sprintf("entry:%d", index), Kind: "folder", Name: folder.Name, Size: "-", ModTime: shortModTime(folder.ModTime), Display: driveDumpRow(folder)}
		builder.WriteString(driveDumpFZFLine(row.Raw, row.Kind, row.Name, row.Size, row.ModTime, row.Display))
		builder.WriteByte('\n')
	}
	args := ui.FZFHub{
		Prompt:       ui.Crown("move") + ui.Muted("> "),
		BorderLabel:  "Move Drive dump",
		Preview:      driveDumpDestinationPreviewCommand(currentPath),
		PreviewLabel: "destination",
		Shortcuts: []ui.FZFShortcut{
			{Label: "Enter", Description: "choose folder"},
			{Label: "Esc", Description: "cancel"},
		},
		ExtraArgs: []string{
			"--delimiter=\t",
			"--with-nth=6..",
			"--nth=2,3,6..",
			"--header-lines=1",
		},
	}.Args()
	output, err := m.Runner.OutputWithInput(ctx, "", []byte(builder.String()), "fzf", args...)
	if err != nil && len(output) == 0 {
		return DriveDumpEntry{}, false, nil
	}
	_, selected := ui.ParseFZFExpectOutput(string(output))
	if len(selected) == 0 {
		return DriveDumpEntry{}, false, nil
	}
	index, ok := parseDriveSelection(driveDumpRawFromLine(selected[0]), len(folders))
	if !ok {
		return DriveDumpEntry{}, false, fmt.Errorf("invalid folder selection: %s", selected[0])
	}
	return folders[index], true, nil
}

func (m *Manager) rcloneRemotes(ctx context.Context) ([]string, error) {
	out, err := m.Runner.Output(ctx, "", "rclone", "listremotes")
	if err != nil {
		return nil, err
	}
	remotes := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		remote := normalizeRcloneRemote(line)
		if remote != "" {
			remotes = append(remotes, remote)
		}
	}
	sort.Strings(remotes)
	return remotes, nil
}

func normalizeRcloneRemote(value string) string {
	return strings.TrimSuffix(strings.TrimSpace(value), ":")
}

func rcloneRemoteExists(remotes []string, remote string) bool {
	remote = normalizeRcloneRemote(remote)
	for _, candidate := range remotes {
		if normalizeRcloneRemote(candidate) == remote {
			return true
		}
	}
	return false
}

func unknownRcloneRemoteError(remote string, remotes []string) error {
	if len(remotes) == 0 {
		return fmt.Errorf("rclone remote %q is not configured; run `rclone config` before downloading dumps", remote)
	}
	return fmt.Errorf("rclone remote %q is not configured; available remotes: %s", remote, strings.Join(remotes, ", "))
}

func (m *Manager) selectImportDatabase(ctx context.Context) (string, error) {
	if err := m.prepareAuth(); err != nil {
		return "", err
	}
	var databases []DatabaseInfo
	err := ui.RunWithRoyalLoader(ui.LoaderOptions{Action: "fetching", Subject: "databases"}, func() error {
		var listErr error
		databases, listErr = m.databaseInfos(ctx)
		return listErr
	})
	if err != nil {
		return "", err
	}
	create := "[+] Create New Database"
	options := append([]DatabaseInfo{{Name: create, SizeKnown: false}}, databases...)
	selected, err := m.selectDatabaseInfos(ctx, "Target database", options, false)
	if err != nil || len(selected) == 0 {
		return "", err
	}
	if selected[0] != create {
		return selected[0], nil
	}
	return ui.Prompt("New database name")
}

func (m *Manager) prepareAuth() error {
	if m.defaultsFile != "" {
		return nil
	}
	if _, err := m.Runner.LookPath("mysql"); err != nil {
		return fmt.Errorf("mysql is required")
	}
	password, err := promptPassword("MySQL password for " + m.Config.Project.DB.User)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp("", "dvv-mysql-*.cnf")
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	lines := []string{"[client]"}
	if m.Config.Project.DB.Host != "" {
		lines = append(lines, "host="+m.Config.Project.DB.Host)
		lines = append(lines, "port="+m.Config.Project.DB.Port)
	}
	lines = append(lines, "user="+m.Config.Project.DB.User)
	lines = append(lines, "password="+password)
	if _, err := file.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	m.defaultsFile = file.Name()
	return nil
}

func (m *Manager) cleanup() {
	if m.defaultsFile != "" {
		_ = os.Remove(m.defaultsFile)
	}
}

func (m *Manager) mysql(ctx context.Context, args ...string) error {
	fullArgs := append([]string{"--defaults-extra-file=" + m.defaultsFile}, args...)
	return run.Quiet(ctx, m.Runner, "", "mysql", fullArgs...)
}

func (m *Manager) ensureDatabase(ctx context.Context, name string, showResult bool) error {
	return ui.RunWithRoyalLoader(ui.LoaderOptions{
		Action:        "preparing",
		Subject:       "database",
		Detail:        name,
		ShowResult:    showResult,
		SuccessAction: "ready",
	}, func() error {
		return m.mysql(ctx, "-e", "CREATE DATABASE IF NOT EXISTS "+identifier(name)+" CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;")
	})
}

func (m *Manager) mysqlOutput(ctx context.Context, args ...string) ([]byte, error) {
	fullArgs := append([]string{"--defaults-extra-file=" + m.defaultsFile}, args...)
	return m.Runner.Output(ctx, "", "mysql", fullArgs...)
}

func (m *Manager) databases(ctx context.Context) ([]string, error) {
	out, err := m.mysqlOutput(ctx, "-N", "-s", "-e", "SHOW DATABASES;")
	if err != nil {
		return nil, err
	}
	excluded := map[string]bool{"information_schema": true, "performance_schema": true, "mysql": true, "sys": true}
	values := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !excluded[line] {
			values = append(values, line)
		}
	}
	sort.Strings(values)
	return values, nil
}

func (m *Manager) databaseInfos(ctx context.Context) ([]DatabaseInfo, error) {
	names, err := m.databases(ctx)
	if err != nil {
		return nil, err
	}
	sizes, err := m.databaseSizeMap(ctx)
	if err != nil {
		infos := make([]DatabaseInfo, 0, len(names))
		for _, name := range names {
			infos = append(infos, DatabaseInfo{Name: name})
		}
		return infos, nil
	}
	infos := make([]DatabaseInfo, 0, len(names))
	for _, name := range names {
		infos = append(infos, DatabaseInfo{Name: name, SizeBytes: sizes[name], SizeKnown: true})
	}
	return infos, nil
}

func (m *Manager) databaseSizeMap(ctx context.Context) (map[string]int64, error) {
	query := "SELECT table_schema, COALESCE(SUM(data_length + index_length), 0) FROM information_schema.tables WHERE table_schema NOT IN ('information_schema', 'performance_schema', 'mysql', 'sys') GROUP BY table_schema;"
	out, err := m.mysqlOutput(ctx, "-N", "-s", "-e", query)
	if err != nil {
		return nil, err
	}
	sizes := map[string]int64{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			fields = strings.Fields(line)
		}
		if len(fields) < 2 {
			continue
		}
		size, err := strconv.ParseInt(strings.TrimSpace(fields[len(fields)-1]), 10, 64)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(strings.Join(fields[:len(fields)-1], " "))
		if name != "" {
			sizes[name] = size
		}
	}
	return sizes, nil
}

func (m *Manager) truncate(ctx context.Context, dbName string) error {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = " + sqlString(dbName) + ";"
	out, err := m.mysqlOutput(ctx, "-N", "-s", "-e", query)
	if err != nil {
		return err
	}
	statements := []string{"SET FOREIGN_KEY_CHECKS = 0;"}
	for _, table := range strings.Split(string(out), "\n") {
		table = strings.TrimSpace(table)
		if table != "" {
			statements = append(statements, "TRUNCATE TABLE "+identifier(table)+";")
		}
	}
	statements = append(statements, "SET FOREIGN_KEY_CHECKS = 1;")
	return m.mysql(ctx, dbName, "-e", strings.Join(statements, " "))
}

func (m *Manager) importDump(ctx context.Context, dump string, dbName string) error {
	info, err := os.Stat(dump)
	if err != nil {
		return err
	}
	if err := m.prepareAuth(); err != nil {
		return err
	}

	file, err := os.Open(dump)
	if err != nil {
		return err
	}
	defer file.Close()

	compressed, err := isGzipFile(file)
	if err != nil {
		return err
	}
	progress := ui.NewRoyalProgressLoader(ui.ProgressOptions{
		Action:  "importing",
		Subject: filepath.Base(dump),
		Detail:  "scanning tables",
		Total:   info.Size(),
	})
	counting := &countingReader{reader: file, onRead: progress.Add}
	var source io.Reader = counting
	if compressed {
		gz, err := gzip.NewReader(counting)
		if err != nil {
			return err
		}
		defer gz.Close()
		source = gz
	}
	sqlReader, removedInvalidLines := sanitizeSQLReader(source)
	sqlReader = newTableTrackingReader(sqlReader, func(table string) {
		progress.SetDetail("table " + table)
	})

	cmd := exec.CommandContext(ctx, "mysql", m.mysqlImportArgs(ctx, dbName)...)
	cmd.Stdin = sqlReader
	cmd.Stdout = os.Stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	progress.Start()
	importOK := false
	defer func() {
		progress.Finish(importOK)
	}()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("import failed: %s", summarizeCommandError(err, stderr.String()))
	}
	importOK = true
	progress.Finish(true)
	if removed := removedInvalidLines(); removed > 0 {
		ui.Warn("Removed %d invalid SQL line(s)", removed)
	}
	ui.OK("Imported %s into %s", filepath.Base(dump), dbName)
	return nil
}

func (m *Manager) mysqlImportArgs(ctx context.Context, dbName string) []string {
	args := []string{
		"--defaults-extra-file=" + m.defaultsFile,
		"--binary-mode=1",
		"--default-character-set=utf8mb4",
	}
	if m.supportsNonStandardFKCompatibility(ctx) {
		args = append(args, "--init-command=SET @@session.restrict_fk_on_non_standard_key=OFF")
	}
	return append(args, dbName)
}

func (m *Manager) supportsNonStandardFKCompatibility(ctx context.Context) bool {
	out, err := m.mysqlOutput(ctx, "-N", "-s", "-e", "SHOW VARIABLES LIKE 'restrict_fk_on_non_standard_key';")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "restrict_fk_on_non_standard_key")
}

func (m *Manager) dumpFiles() ([]string, error) {
	if err := os.MkdirAll(m.Config.Project.DB.DumpsDir, 0o755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(m.Config.Project.DB.DumpsDir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		name := entry.Name()
		if isDumpFile(filepath.Join(m.Config.Project.DB.DumpsDir, name), name) {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, nil
}

func (m *Manager) dumpFileInfos() ([]DumpFile, error) {
	files, err := m.dumpFiles()
	if err != nil {
		return nil, err
	}
	dumps := make([]DumpFile, 0, len(files))
	for _, name := range files {
		dump := DumpFile{Name: name}
		if info, err := os.Stat(filepath.Join(m.Config.Project.DB.DumpsDir, name)); err == nil {
			dump.SizeBytes = info.Size()
			dump.SizeKnown = true
		}
		dumps = append(dumps, dump)
	}
	return dumps, nil
}

func normalizeDownloadDumpFileName(fileName string, fallbackID string) string {
	name := filepath.Base(strings.TrimSpace(fileName))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = strings.TrimSpace(fallbackID)
	}
	if name == "" {
		name = "dump"
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".sql.gz") || strings.HasSuffix(lower, ".gz") {
		return name
	}
	return name + ".sql.gz"
}

func isDumpFile(path string, name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".sql.gz") {
		return true
	}
	if fileHasGzipHeader(path) {
		return true
	}
	return fileLooksLikeSQLDump(path)
}

func fileHasGzipHeader(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	ok, err := isGzipFile(file)
	return err == nil && ok
}

func isGzipFile(file *os.File) (bool, error) {
	current, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}
	header := make([]byte, 2)
	n, readErr := file.Read(header)
	if _, err := file.Seek(current, io.SeekStart); err != nil {
		return false, err
	}
	if readErr != nil && readErr != io.EOF {
		return false, readErr
	}
	return n == 2 && header[0] == 0x1f && header[1] == 0x8b, nil
}

func fileLooksLikeSQLDump(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false
	}
	if n == 0 || bytes.IndexByte(buffer[:n], 0) >= 0 {
		return false
	}

	sample := strings.TrimSpace(strings.TrimPrefix(string(buffer[:n]), "\xef\xbb\xbf"))
	sample = strings.ToLower(sample)
	for _, prefix := range []string{"--", "/*", "create ", "insert ", "drop ", "set ", "lock tables", "delimiter ", "use "} {
		if strings.HasPrefix(sample, prefix) {
			return true
		}
	}
	return false
}

type countingReader struct {
	reader io.Reader
	onRead func(int)
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.onRead != nil {
		r.onRead(n)
	}
	return n, err
}

type tableTrackingReader struct {
	reader  io.Reader
	onTable func(string)
	buffer  string
	current string
}

func newTableTrackingReader(reader io.Reader, onTable func(string)) io.Reader {
	if onTable == nil {
		return reader
	}
	return &tableTrackingReader{reader: reader, onTable: onTable}
}

func (r *tableTrackingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.consume(string(p[:n]))
	}
	if err == io.EOF && r.buffer != "" {
		r.consumeLine(r.buffer)
		r.buffer = ""
	}
	return n, err
}

func (r *tableTrackingReader) consume(chunk string) {
	r.buffer += chunk
	for {
		index := strings.IndexByte(r.buffer, '\n')
		if index < 0 {
			break
		}
		r.consumeLine(r.buffer[:index])
		r.buffer = r.buffer[index+1:]
	}
	if len(r.buffer) > 64*1024 {
		r.consumeLine(r.buffer)
		r.buffer = ""
	}
}

func (r *tableTrackingReader) consumeLine(line string) {
	table := detectSQLTable(line)
	if table == "" || table == r.current {
		return
	}
	r.current = table
	r.onTable(table)
}

func detectSQLTable(line string) string {
	line = strings.TrimSpace(strings.TrimPrefix(line, "\xef\xbb\xbf"))
	if line == "" {
		return ""
	}
	line = unwrapMySQLVersionedComment(line)
	for _, pattern := range sqlTablePatterns {
		matches := pattern.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}
		return extractTableIdentifier(matches[1])
	}
	return ""
}

func unwrapMySQLVersionedComment(line string) string {
	if !strings.HasPrefix(line, "/*!") {
		return line
	}
	index := 3
	for index < len(line) && line[index] >= '0' && line[index] <= '9' {
		index++
	}
	if index == 3 {
		return line
	}
	body := strings.TrimSpace(line[index:])
	body = strings.TrimSuffix(body, ";")
	body = strings.TrimSpace(strings.TrimSuffix(body, "*/"))
	return strings.TrimSpace(body)
}

func extractTableIdentifier(value string) string {
	rest := strings.TrimSpace(value)
	last := ""
	for {
		token, remaining := popSQLIdentifier(rest)
		if token == "" {
			return ""
		}
		last = token
		rest = strings.TrimSpace(remaining)
		if !strings.HasPrefix(rest, ".") {
			break
		}
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "."))
	}
	return cleanSQLIdentifier(last)
}

func popSQLIdentifier(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	quote := value[0]
	if quote == '`' || quote == '"' || quote == '\'' {
		for index := 1; index < len(value); index++ {
			if value[index] == quote {
				if index+1 < len(value) && value[index+1] == quote {
					index++
					continue
				}
				return value[:index+1], value[index+1:]
			}
		}
		return "", value
	}
	for index, char := range value {
		if unicode.IsSpace(char) || char == '(' || char == ',' || char == ';' {
			return value[:index], value[index:]
		}
	}
	return value, ""
}

func cleanSQLIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '`' && last == '`') || (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			value = value[1 : len(value)-1]
			value = strings.ReplaceAll(value, string([]byte{first, first}), string(first))
		}
	}
	return value
}

func sanitizeSQLReader(source io.Reader) (io.Reader, func() int64) {
	reader, writer := io.Pipe()
	var removed atomic.Int64
	go func() {
		scanner := bufio.NewScanner(source)
		scanner.Buffer(make([]byte, 64*1024), 64*1024*1024)
		for scanner.Scan() {
			line := strings.TrimSuffix(scanner.Text(), "\r")
			if line == "-" {
				removed.Add(1)
				continue
			}
			if _, err := io.WriteString(writer, line+"\n"); err != nil {
				_ = writer.CloseWithError(err)
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()
	return reader, removed.Load
}

func summarizeCommandError(err error, stderr string) string {
	message := strings.TrimSpace(stderr)
	if message == "" {
		return err.Error()
	}
	lines := strings.Split(message, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return err.Error()
	}
	if strings.Contains(last, "ERROR 6125") || strings.Contains(last, "Missing unique key") {
		return last + "; MySQL is rejecting a foreign key that references a non-unique key. dvv enables restrict_fk_on_non_standard_key=OFF when the server supports it."
	}
	return last
}

func promptPassword(label string) (string, error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return ui.Prompt(label)
	}
	defer tty.Close()

	fmt.Fprintf(tty, "%s %s: ", ui.Crown("dvv"), label)
	_ = runStty(tty, "-echo")
	defer func() {
		_ = runStty(tty, "echo")
		fmt.Fprintln(tty)
	}()
	reader := bufio.NewReader(tty)
	value, err := reader.ReadString('\n')
	return strings.TrimSpace(value), err
}

func runStty(tty *os.File, arg string) error {
	cmd := exec.Command("stty", arg)
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	return cmd.Run()
}

func identifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func sqlString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `''`)
	return "'" + value + "'"
}

func dbActionHeader() string {
	return fmt.Sprintf(" %s  %s", ui.Crown("NO"), ui.Crown(fixedWidth("ACTION", 24)))
}

func dbActionRow(index int, action Action) string {
	return fmt.Sprintf("%s  %s", ui.Muted(fmt.Sprintf("%02d", index+1)), ui.Accent(fixedWidth(action.Label, 24)))
}

func dbActionLine(raw string, label string, description string, display string) string {
	return strings.Join([]string{
		cleanFZFField(raw),
		cleanFZFField(label),
		cleanFZFField(description),
		display,
	}, "\t")
}

func cleanFZFField(value string) string {
	value = strings.ReplaceAll(value, "\t", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func dbActionPreviewCommand() string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
label=$(printf "%s" "$line" | cut -f2)
description=$(printf "%s" "$line" | cut -f3)
printf "%sDatabase action%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-8s%s %s\n" "$dvv_label" "Action" "$dvv_reset" "$label"
printf "  %s%-8s%s %s\n" "$dvv_label" "Command" "$dvv_reset" "$raw"
printf "\n%sWhat it does%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%s%s\n" "$dvv_muted" "$description" "$dvv_reset"
' sh {}`
}

func databaseNames(databases []DatabaseInfo) []string {
	names := make([]string, 0, len(databases))
	for _, database := range databases {
		names = append(names, database.Name)
	}
	return names
}

func databaseSize(database DatabaseInfo) string {
	if !database.SizeKnown {
		return "unknown"
	}
	return human.FormatBytes(database.SizeBytes)
}

func databaseInfoHeader() string {
	return fmt.Sprintf(" %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("DATABASE", 32)),
		ui.Crown("SIZE"),
	)
}

func databaseInfoRow(index int, database DatabaseInfo) string {
	return fmt.Sprintf("%s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(database.Name, 32)),
		ui.Muted(databaseSize(database)),
	)
}

func dumpFileSize(dump DumpFile) string {
	if !dump.SizeKnown {
		return "-"
	}
	return human.FormatBytes(dump.SizeBytes)
}

func dumpFileHeader() string {
	return fmt.Sprintf(" %s  %s  %s",
		ui.Crown("NO"),
		ui.Crown(fixedWidth("DUMP", 42)),
		ui.Crown("SIZE"),
	)
}

func dumpFileRow(index int, dump DumpFile) string {
	return fmt.Sprintf("%s  %s  %s",
		ui.Muted(fmt.Sprintf("%02d", index+1)),
		ui.Accent(fixedWidth(dump.Name, 42)),
		ui.Muted(dumpFileSize(dump)),
	)
}

func driveDumpSelectionRows(currentPath string, entries []DriveDumpEntry) []driveDumpSelectionRow {
	rows := make([]driveDumpSelectionRow, 0, len(entries)+1)
	if strings.TrimSpace(currentPath) != "" {
		rows = append(rows, driveDumpSelectionRow{Raw: "__parent__", Kind: "folder", Name: "..", Size: "-", ModTime: "parent", Display: driveDumpParentRow()})
	}
	for index, entry := range entries {
		kind := "file"
		size := human.FormatBytes(entry.Size)
		if entry.IsDir {
			kind = "folder"
			size = "-"
		}
		rows = append(rows, driveDumpSelectionRow{
			Raw:     fmt.Sprintf("entry:%d", index),
			Kind:    kind,
			Name:    entry.Name,
			Size:    size,
			ModTime: shortModTime(entry.ModTime),
			Display: driveDumpRow(entry),
		})
	}
	return rows
}

func driveDumpFZFLine(raw string, kind string, name string, size string, modTime string, display string) string {
	return strings.Join([]string{
		cleanFZFField(raw),
		cleanFZFField(kind),
		cleanFZFField(name),
		cleanFZFField(size),
		cleanFZFField(modTime),
		display,
	}, "\t")
}

func driveDumpHeader() string {
	return fmt.Sprintf(" %s  %s  %s",
		ui.Crown(fixedWidth("TYPE", 8)),
		ui.Crown(fixedWidth("SIZE", 10)),
		ui.Crown(fixedWidth("NAME", driveDumpNameWidth)),
	)
}

func driveDumpParentRow() string {
	return fmt.Sprintf("%s  %s  %s",
		ui.Gold(fixedWidth("folder", 8)),
		ui.Muted(fixedWidth("-", 10)),
		ui.Accent(fixedWidth("..", driveDumpNameWidth)),
	)
}

func driveDumpRow(entry DriveDumpEntry) string {
	kind := "file"
	size := human.FormatBytes(entry.Size)
	if entry.IsDir {
		kind = "folder"
		size = "-"
	}
	return fmt.Sprintf("%s  %s  %s",
		ui.Gold(fixedWidth(kind, 8)),
		ui.Muted(fixedWidth(size, 10)),
		ui.Accent(fixedWidth(compactWidth(entry.Name, driveDumpNameWidth), driveDumpNameWidth)),
	)
}

func driveDumpPreviewCommand(currentPath string) string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
raw=$(printf "%s" "$line" | cut -f1)
kind=$(printf "%s" "$line" | cut -f2)
name=$(printf "%s" "$line" | cut -f3)
size=$(printf "%s" "$line" | cut -f4)
modified=$(printf "%s" "$line" | cut -f5)
printf "%sGoogle Drive dump%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-9s%s ` + shellQuote(dbDefaultString(currentPath, "/")) + `\n" "$dvv_label" "Folder" "$dvv_reset"
printf "\n%sCommands%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "Tab" "$dvv_reset" "$dvv_muted" "mark files" "$dvv_reset"
printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "Shift+M" "$dvv_reset" "$dvv_muted" "move marked files" "$dvv_reset"
printf "  %s[%-7s]%s %s%s%s\n" "$dvv_status" "Shift+R" "$dvv_reset" "$dvv_muted" "refresh folder" "$dvv_reset"
printf "\n%sSelected%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-9s%s %s\n" "$dvv_label" "Type" "$dvv_reset" "$kind"
printf "  %s%-9s%s %s\n" "$dvv_label" "Name" "$dvv_reset" "$name"
printf "  %s%-9s%s %s\n" "$dvv_label" "Size" "$dvv_reset" "$size"
printf "  %s%-9s%s %s\n" "$dvv_label" "Modified" "$dvv_reset" "$modified"
if [ "$raw" = "__parent__" ]; then
  printf "\n%sPress Enter to go to the parent folder.%s\n" "$dvv_muted" "$dvv_reset"
elif [ "$kind" = "folder" ]; then
  printf "\n%sPress Enter to open this folder.%s\n" "$dvv_muted" "$dvv_reset"
else
  printf "\n%sPress Enter to download and import this dump.%s\n" "$dvv_muted" "$dvv_reset"
fi
' sh {}`
}

func driveDumpDestinationPreviewCommand(currentPath string) string {
	return `sh -c '` + ui.FZFPreviewShellPrefix() + `line=$1
name=$(printf "%s" "$line" | cut -f3)
printf "%sMove destination%s\n" "$dvv_heading" "$dvv_reset"
printf "  %s%-9s%s ` + shellQuote(dbDefaultString(currentPath, "/")) + `\n" "$dvv_label" "From" "$dvv_reset"
printf "  %s%-9s%s %s\n" "$dvv_label" "To" "$dvv_reset" "$name"
printf "\n%sPress Enter to choose this folder.%s\n" "$dvv_muted" "$dvv_reset"
' sh {}`
}

func parseDriveSelection(raw string, length int) (int, bool) {
	value := strings.TrimPrefix(strings.TrimSpace(raw), "entry:")
	index, err := strconv.Atoi(value)
	return index, err == nil && index >= 0 && index < length
}

func drivePathJoin(base string, name string) string {
	base = strings.Trim(strings.TrimSpace(base), "/")
	name = strings.Trim(strings.TrimSpace(name), "/")
	if base == "" {
		return name
	}
	if name == "" {
		return base
	}
	return pathpkg.Join(base, name)
}

func isDriveDumpFileName(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(lower, ".sql") || strings.HasSuffix(lower, ".sql.gz") || strings.HasSuffix(lower, ".gz")
}

func shortModTime(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= len("2006-01-02 15:04") {
		value = strings.ReplaceAll(value[:len("2006-01-02T15:04")], "T", " ")
	}
	if value == "" {
		return "-"
	}
	return value
}

func dbDefaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func showHelp() {
	ui.Title("Database Hub")
	fmt.Printf("  %s dvv db\n\n", ui.Bold("Usage:"))
	helpSection("Hub")
	helpEntry("dvv db", "Open the interactive database hub")
	helpEntry("dvv db create [name]", "Create a database")
	helpEntry("dvv db import", "Import a local or Google Drive dump")
	helpEntry("dvv db clean", "Delete local dump files")
	helpEntry("dvv db truncate", "Truncate selected databases")
	helpEntry("dvv db drop", "Drop selected databases")
}

func helpSection(title string) {
	fmt.Printf("  %s\n", ui.Cyan(title+":"))
}

func helpEntry(command string, description string) {
	fmt.Printf("    %-28s %s\n", ui.Bold(command), description)
}

func extractDriveFileID(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	for _, marker := range []string{"/file/d/", "id="} {
		if index := strings.Index(raw, marker); index >= 0 {
			value := raw[index+len(marker):]
			value = strings.TrimLeft(value, "/")
			parts := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '?' || r == '&' })
			if len(parts) == 0 {
				return "", false
			}
			value = parts[0]
			return value, value != ""
		}
	}
	return raw, len(raw) >= 10
}

func extractDriveFolderID(raw string) (string, bool) {
	folder, ok := extractDriveFolderRef(raw)
	return folder.ID, ok
}

func extractDriveFolderRef(raw string) (driveFolderRef, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return driveFolderRef{}, false
	}
	resourceKey := ""
	if parsed, err := url.Parse(raw); err == nil && parsed.RawQuery != "" {
		resourceKey = strings.TrimSpace(parsed.Query().Get("resourcekey"))
	}
	for _, marker := range []string{"/folders/", "folderId=", "folderid="} {
		if index := strings.Index(raw, marker); index >= 0 {
			value := raw[index+len(marker):]
			value = strings.TrimLeft(value, "/")
			parts := strings.FieldsFunc(value, func(r rune) bool { return r == '/' || r == '?' || r == '&' })
			if len(parts) == 0 {
				return driveFolderRef{}, false
			}
			value = parts[0]
			return driveFolderRef{ID: value, ResourceKey: resourceKey}, value != ""
		}
	}
	return driveFolderRef{ID: raw}, len(raw) >= 10
}

func valuesByIndexes(values []string, input string) []string {
	selected := []string{}
	for _, part := range strings.FieldsFunc(input, func(r rune) bool { return r == ',' || r == ' ' }) {
		index, ok := parseIndex(part, len(values))
		if ok {
			selected = append(selected, values[index])
		}
	}
	return selected
}

func nonEmptyRawLines(output string) []string {
	out := []string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		raw := ui.FZFSelectedRaw(line)
		if raw != "" && raw != "__dvv_header__" {
			out = append(out, raw)
		}
	}
	return out
}

func parseIndex(value string, length int) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	var number int
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, false
		}
		number = number*10 + int(r-'0')
	}
	index := number - 1
	return index, index >= 0 && index < length
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func fixedWidth(value string, width int) string {
	value = strings.TrimSpace(value)
	if len(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(value))
}

func compactWidth(value string, width int) string {
	value = strings.TrimSpace(value)
	if width <= 3 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-3]) + "..."
}
