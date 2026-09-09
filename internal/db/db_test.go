package db

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
	"github.com/VitorAllux/devtools/internal/ui"
)

func TestExtractDriveFileID(t *testing.T) {
	tests := map[string]string{
		"https://drive.google.com/file/d/abc123XYZ/view":      "abc123XYZ",
		"https://drive.google.com/open?id=file-123&usp=share": "file-123",
		"raw-file-id-123": "raw-file-id-123",
	}

	for input, expected := range tests {
		got, ok := extractDriveFileID(input)
		if !ok || got != expected {
			t.Fatalf("extractDriveFileID(%q) = %q, %v; want %q, true", input, got, ok, expected)
		}
	}
}

func TestExtractDriveFileIDRejectsIncompleteURL(t *testing.T) {
	for _, input := range []string{"", "https://drive.google.com/file/d/", "https://drive.google.com/open?id="} {
		got, ok := extractDriveFileID(input)
		if ok || got != "" {
			t.Fatalf("extractDriveFileID(%q) = %q, %v; want empty false", input, got, ok)
		}
	}
}

func TestSQLQuoting(t *testing.T) {
	if got := identifier("my-db`name"); got != "`my-db``name`" {
		t.Fatalf("identifier = %q", got)
	}
	if got := sqlString("elo'full"); got != "'elo''full'" {
		t.Fatalf("sqlString = %q", got)
	}
}

func TestDumpFilesAreSortedAndFiltered(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.sql.gz", "a.sql", "notes.txt", "c.dump.gz"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "adami"), []byte{0x1f, 0x8b, 0x08}, 0o600); err != nil {
		t.Fatalf("WriteFile gzip dump failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plain-dump"), []byte("-- MySQL dump\nCREATE TABLE users (id int);\n"), 0o600); err != nil {
		t.Fatalf("WriteFile SQL dump failed: %v", err)
	}
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	cfg.Project.DB.DumpsDir = dir
	manager := Manager{Config: cfg}

	got, err := manager.dumpFiles()
	if err != nil {
		t.Fatalf("dumpFiles returned error: %v", err)
	}
	want := []string{"a.sql", "adami", "b.sql.gz", "plain-dump"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dumpFiles = %#v, want %#v", got, want)
	}
}

func TestNormalizeDownloadDumpFileNameAddsDefaultExtension(t *testing.T) {
	tests := map[string]string{
		"":             "file-id.sql.gz",
		"adami":        "adami.sql.gz",
		"adami.sql":    "adami.sql",
		"adami.sql.gz": "adami.sql.gz",
		"adami.gz":     "adami.gz",
		"../adami":     "adami.sql.gz",
	}
	for input, expected := range tests {
		got := normalizeDownloadDumpFileName(input, "file-id")
		if got != expected {
			t.Fatalf("normalizeDownloadDumpFileName(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestIsGzipFileUsesContentHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dump-without-extension")
	if err := os.WriteFile(path, []byte{0x1f, 0x8b, 0x08}, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer file.Close()

	got, err := isGzipFile(file)
	if err != nil {
		t.Fatalf("isGzipFile returned error: %v", err)
	}
	if !got {
		t.Fatal("expected gzip header to be detected")
	}
	if offset, err := file.Seek(0, io.SeekCurrent); err != nil || offset != 0 {
		t.Fatalf("file offset = %d, err=%v; want 0, nil", offset, err)
	}
}

func TestSanitizeSQLReaderRemovesStandaloneDashLines(t *testing.T) {
	reader, removed := sanitizeSQLReader(strings.NewReader("select 1;\r\n-\nselect '-';\n"))
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll returned error: %v", err)
	}
	want := "select 1;\nselect '-';\n"
	if string(got) != want {
		t.Fatalf("sanitized SQL = %q, want %q", got, want)
	}
	if removed() != 1 {
		t.Fatalf("removed = %d, want 1", removed())
	}
}

func TestDetectSQLTableNames(t *testing.T) {
	tests := map[string]string{
		"-- Table structure for table `users`":              "users",
		"-- Dumping data for table `audit_logs`":            "audit_logs",
		"CREATE TABLE IF NOT EXISTS `app`.`orders` (":       "orders",
		"CREATE TABLE `weird``name` (`id` int);":            "weird`name",
		"INSERT IGNORE INTO plain_table (`id`) VALUES (1);": "plain_table",
		"REPLACE INTO `tenant_users` VALUES (1);":           "tenant_users",
		"ALTER TABLE \"events\" ADD KEY `idx` (`id`);":      "events",
		"/*!40000 ALTER TABLE `events` DISABLE KEYS */;":    "events",
		"DROP TABLE IF EXISTS 'old_table';":                 "old_table",
		"LOCK TABLES `user.sessions` WRITE;":                "user.sessions",
		"select * from users;":                              "",
	}

	for input, expected := range tests {
		if got := detectSQLTable(input); got != expected {
			t.Fatalf("detectSQLTable(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestTableTrackingReaderReportsTableChanges(t *testing.T) {
	reported := []string{}
	reader := newTableTrackingReader(strings.NewReader(strings.Join([]string{
		"-- Table structure for table `users`",
		"CREATE TABLE `users` (`id` int);",
		"INSERT INTO `orders` (`id`) VALUES (1);",
		"INSERT INTO `orders` (`id`) VALUES (2);",
		"ALTER TABLE `invoices` ADD KEY `id` (`id`);",
	}, "\n")), func(table string) {
		reported = append(reported, table)
	})

	buffer := make([]byte, 7)
	for {
		_, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read returned error: %v", err)
		}
	}

	want := []string{"users", "orders", "invoices"}
	if !reflect.DeepEqual(reported, want) {
		t.Fatalf("reported tables = %#v, want %#v", reported, want)
	}
}

func TestEnsureDatabaseRunsExpectedMysqlCommand(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &dbFakeRunner{}
	manager := &Manager{
		Config:       &config.Config{Project: config.DefaultProjectConfig()},
		Runner:       runner,
		defaultsFile: "/tmp/client.cnf",
	}

	if err := manager.ensureDatabase(context.Background(), "my-db", false); err != nil {
		t.Fatalf("ensureDatabase returned error: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("calls = %#v, want one mysql call", runner.calls)
	}
	call := runner.calls[0]
	if call.name != "mysql" {
		t.Fatalf("command = %q, want mysql", call.name)
	}
	wantArgs := []string{
		"--defaults-extra-file=/tmp/client.cnf",
		"-e",
		"CREATE DATABASE IF NOT EXISTS `my-db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
	}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", call.args, wantArgs)
	}
}

func TestCommandCreateUsesProvidedName(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	runner := &dbFakeRunner{}
	manager := &Manager{
		Config:       &config.Config{Project: config.DefaultProjectConfig()},
		Runner:       runner,
		defaultsFile: "/tmp/client.cnf",
	}

	if err := manager.CommandCreate(context.Background(), []string{"reporting"}); err != nil {
		t.Fatalf("CommandCreate returned error: %v", err)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "mysql" {
		t.Fatalf("calls = %#v, want one mysql create command", runner.calls)
	}
	if !strings.Contains(strings.Join(runner.calls[0].args, " "), "CREATE DATABASE IF NOT EXISTS `reporting`") {
		t.Fatalf("args = %#v", runner.calls[0].args)
	}
}

func TestDatabasesFiltersSystemSchemasAndSorts(t *testing.T) {
	runner := &dbFakeRunner{outputs: []string{"sys\nzeta\ninformation_schema\napp\nmysql\nperformance_schema\n"}}
	manager := &Manager{
		Config:       &config.Config{Project: config.DefaultProjectConfig()},
		Runner:       runner,
		defaultsFile: "/tmp/client.cnf",
	}

	got, err := manager.databases(context.Background())
	if err != nil {
		t.Fatalf("databases returned error: %v", err)
	}
	want := []string{"app", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("databases = %#v, want %#v", got, want)
	}
}

func TestTruncateBuildsForeignKeySafeStatements(t *testing.T) {
	runner := &dbFakeRunner{outputs: []string{"users\norders\n", ""}}
	manager := &Manager{
		Config:       &config.Config{Project: config.DefaultProjectConfig()},
		Runner:       runner,
		defaultsFile: "/tmp/client.cnf",
	}

	if err := manager.truncate(context.Background(), "dev-db"); err != nil {
		t.Fatalf("truncate returned error: %v", err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("calls = %#v, want metadata query and truncate command", runner.calls)
	}
	call := runner.calls[1]
	wantArgs := []string{
		"--defaults-extra-file=/tmp/client.cnf",
		"dev-db",
		"-e",
		"SET FOREIGN_KEY_CHECKS = 0; TRUNCATE TABLE `users`; TRUNCATE TABLE `orders`; SET FOREIGN_KEY_CHECKS = 1;",
	}
	if !reflect.DeepEqual(call.args, wantArgs) {
		t.Fatalf("truncate args = %#v, want %#v", call.args, wantArgs)
	}
}

func TestDropDatabaseCommandUsesIdentifierQuoting(t *testing.T) {
	runner := &dbFakeRunner{}
	manager := &Manager{
		Config:       &config.Config{Project: config.DefaultProjectConfig()},
		Runner:       runner,
		defaultsFile: "/tmp/client.cnf",
	}

	if err := manager.mysql(context.Background(), "-e", "DROP DATABASE "+identifier("client-db")+";"); err != nil {
		t.Fatalf("mysql returned error: %v", err)
	}
	wantArgs := []string{"--defaults-extra-file=/tmp/client.cnf", "-e", "DROP DATABASE `client-db`;"}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("drop args = %#v, want %#v", runner.calls[0].args, wantArgs)
	}
}

func TestCommandCleanDeletesSelectedDumpAfterConfirmation(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "1")

	dir := t.TempDir()
	selected := filepath.Join(dir, "old.sql")
	kept := filepath.Join(dir, "keep.sql")
	if err := os.WriteFile(selected, []byte("select 1;"), 0o600); err != nil {
		t.Fatalf("WriteFile selected failed: %v", err)
	}
	if err := os.WriteFile(kept, []byte("select 2;"), 0o600); err != nil {
		t.Fatalf("WriteFile kept failed: %v", err)
	}

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	originalStdin := os.Stdin
	os.Stdin = stdinReader
	t.Cleanup(func() {
		os.Stdin = originalStdin
		stdinReader.Close()
	})
	if _, err := stdinWriter.WriteString("y\n"); err != nil {
		t.Fatalf("WriteString stdin failed: %v", err)
	}
	stdinWriter.Close()

	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	cfg.Project.DB.DumpsDir = dir
	runner := &dbFakeRunner{fzfOutput: []byte(ui.FZFHiddenRow("old.sql", "old.sql") + "\n")}
	manager := &Manager{Config: cfg, Runner: runner}

	if err := manager.CommandClean(context.Background()); err != nil {
		t.Fatalf("CommandClean returned error: %v", err)
	}
	if _, err := os.Stat(selected); !os.IsNotExist(err) {
		t.Fatalf("selected dump should be deleted, stat err = %v", err)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("kept dump should remain, stat err = %v", err)
	}
}

func TestDownloadDumpRejectsUnknownRcloneRemote(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "1")
	withDBStdin(t, "gm\n")

	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	cfg.Project.DB.DumpsDir = t.TempDir()
	runner := &dbFakeRunner{outputs: []string{"gdrive:\nbackup:\n"}}
	manager := &Manager{Config: cfg, Runner: runner}

	_, err := manager.downloadDump(context.Background())
	if err == nil {
		t.Fatal("downloadDump should reject unknown rclone remote")
	}
	want := `rclone remote "gm" is not configured; available remotes: backup, gdrive`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
	if len(runner.calls) != 1 || runner.calls[0].name != "rclone" || !reflect.DeepEqual(runner.calls[0].args, []string{"listremotes"}) {
		t.Fatalf("calls = %#v, want only rclone listremotes", runner.calls)
	}
}

func TestImportDumpUsesMysqlBinaryModeAndSanitizer(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "1")

	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	stdinFile := filepath.Join(dir, "stdin")
	mysql := filepath.Join(dir, "mysql")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$DVV_MYSQL_ARGS_FILE\"\ncat > \"$DVV_MYSQL_STDIN_FILE\"\n"
	if err := os.WriteFile(mysql, []byte(script), 0o700); err != nil {
		t.Fatalf("WriteFile mysql fake failed: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("DVV_MYSQL_ARGS_FILE", argsFile)
	t.Setenv("DVV_MYSQL_STDIN_FILE", stdinFile)

	dump := filepath.Join(dir, "adami")
	if err := os.WriteFile(dump, []byte("select 1;\n-\ninsert into logs values ('-');\n"), 0o600); err != nil {
		t.Fatalf("WriteFile dump failed: %v", err)
	}
	manager := &Manager{
		Config:       &config.Config{Project: config.DefaultProjectConfig()},
		Runner:       &dbFakeRunner{},
		defaultsFile: "/tmp/client.cnf",
	}

	if err := manager.importDump(context.Background(), dump, "adami"); err != nil {
		t.Fatalf("importDump returned error: %v", err)
	}

	argsContent, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("ReadFile args failed: %v", err)
	}
	wantArgs := "--defaults-extra-file=/tmp/client.cnf\n--binary-mode=1\n--default-character-set=utf8mb4\nadami\n"
	if string(argsContent) != wantArgs {
		t.Fatalf("mysql args = %q, want %q", argsContent, wantArgs)
	}
	stdinContent, err := os.ReadFile(stdinFile)
	if err != nil {
		t.Fatalf("ReadFile stdin failed: %v", err)
	}
	wantSQL := "select 1;\ninsert into logs values ('-');\n"
	if string(stdinContent) != wantSQL {
		t.Fatalf("mysql stdin = %q, want %q", stdinContent, wantSQL)
	}
}

func TestSelectionParsingHelpers(t *testing.T) {
	values := []string{"one", "two", "three"}
	if got := valuesByIndexes(values, "1, 3 bad 9"); !reflect.DeepEqual(got, []string{"one", "three"}) {
		t.Fatalf("valuesByIndexes = %#v", got)
	}
	lines := ui.FZFHiddenRow("alpha", "Alpha") + "\n" + ui.FZFHiddenHeader("Header") + "\nplain\n"
	if got := nonEmptyRawLines(lines); !reflect.DeepEqual(got, []string{"alpha", "plain"}) {
		t.Fatalf("nonEmptyRawLines = %#v", got)
	}
	if index, ok := parseIndex("2", 3); !ok || index != 1 {
		t.Fatalf("parseIndex = %d ok=%v", index, ok)
	}
	for _, value := range []string{"", "0", "4", "x"} {
		if _, ok := parseIndex(value, 3); ok {
			t.Fatalf("parseIndex(%q) should be invalid", value)
		}
	}
	if shellQuote("a'b") != "'a'\\''b'" {
		t.Fatal("shellQuote did not escape single quotes")
	}
	if got := fixedWidth("abcdef", 4); got != "abcdef" {
		t.Fatalf("fixedWidth = %q", got)
	}
	if got := fixedWidth("ab", 4); got != "ab  " {
		t.Fatalf("fixedWidth padding = %q", got)
	}
}

func TestDBHubFormattingHelpers(t *testing.T) {
	action := Action{Name: "import", Label: "Import dump", Description: "Import a local or Google Drive SQL dump"}
	if header := dbActionHeader(); !strings.Contains(header, "ACTION") || strings.Contains(header, "DETAIL") {
		t.Fatalf("header = %q", header)
	}
	if row := dbActionRow(0, action); !strings.Contains(row, "01") || !strings.Contains(row, "Import dump") {
		t.Fatalf("row = %q", row)
	} else if strings.Contains(row, action.Description) {
		t.Fatalf("visible action row should keep description in preview only: %q", row)
	}
	line := dbActionLine(action.Name, action.Label, action.Description, dbActionRow(0, action))
	if !strings.Contains(line, action.Description) {
		t.Fatalf("hidden action line should keep description for preview: %q", line)
	}
	preview := dbActionPreviewCommand()
	for _, want := range []string{"Action", "Command", "What it does", "description="} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q: %s", want, preview)
		}
	}
}

func TestSummarizeCommandErrorUsesLastStderrLine(t *testing.T) {
	err := errors.New("exit status 1")
	got := summarizeCommandError(err, "warning\nERROR 1064 near broken\n")
	if got != "ERROR 1064 near broken" {
		t.Fatalf("summary = %q", got)
	}
	if fallback := summarizeCommandError(err, ""); fallback != "exit status 1" {
		t.Fatalf("fallback = %q", fallback)
	}
}

type dbCommandCall struct {
	dir  string
	name string
	args []string
}

type dbFakeRunner struct {
	calls     []dbCommandCall
	outputs   []string
	paths     map[string]bool
	fzfOutput []byte
	fzfErr    error
}

func (r *dbFakeRunner) Run(_ context.Context, dir string, name string, args ...string) error {
	r.calls = append(r.calls, dbCommandCall{dir: dir, name: name, args: append([]string{}, args...)})
	return nil
}

func (r *dbFakeRunner) Output(_ context.Context, dir string, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, dbCommandCall{dir: dir, name: name, args: append([]string{}, args...)})
	if len(r.outputs) == 0 {
		return nil, nil
	}
	out := r.outputs[0]
	r.outputs = r.outputs[1:]
	return []byte(out), nil
}

func (r *dbFakeRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	if r.fzfOutput != nil || r.fzfErr != nil {
		return r.fzfOutput, r.fzfErr
	}
	return nil, errors.New("unexpected output with input command")
}

func (r *dbFakeRunner) Start(context.Context, string, string, ...string) error {
	return errors.New("unexpected start command")
}

func (r *dbFakeRunner) LookPath(name string) (string, error) {
	if r.paths != nil && !r.paths[name] {
		return "", errors.New("not found")
	}
	return "/usr/bin/" + name, nil
}

func withDBStdin(t *testing.T, input string) {
	t.Helper()
	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	originalStdin := os.Stdin
	os.Stdin = stdinReader
	t.Cleanup(func() {
		os.Stdin = originalStdin
		stdinReader.Close()
	})
	if _, err := stdinWriter.WriteString(input); err != nil {
		t.Fatalf("WriteString stdin failed: %v", err)
	}
	stdinWriter.Close()
}
