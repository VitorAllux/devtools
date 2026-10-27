package ui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

func TestRoyalLoadersFitTerminalWidth(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	originalSize := terminalSize
	t.Cleanup(func() { terminalSize = originalSize })
	for _, noColor := range []string{"", "1"} {
		t.Run("NO_COLOR="+noColor, func(t *testing.T) {
			t.Setenv("NO_COLOR", noColor)
			for _, columns := range []int{1, 12, 40, 80, 120, 200} {
				t.Run(fmt.Sprint(columns), func(t *testing.T) {
					terminalSize = func(int) (int, int, error) { return columns, 24, nil }
					for _, subject := range []string{
						"sample_production_dump_2026-10-20_10-48-51.sql.gz",
						strings.Repeat("資料👩‍💻e\u0301", 20) + ".sql.gz",
					} {
						options := ProgressOptions{
							Action:  "importing",
							Subject: subject,
							Detail:  "table contract_closing_transaction_records",
						}
						for _, status := range []progressStatus{progressRunning, progressSuccess, progressFailure} {
							percent := 42
							if status == progressSuccess {
								percent = 100
							}
							frame := renderRoyalProgressFrame(options, percent, status)
							assertLoaderFrameFits(t, frame, columns)
							if columns >= 12 && !strings.HasSuffix(ansi.Strip(frame), fmt.Sprintf("%d%%", percent)) {
								t.Fatalf("percentage missing from %q", frame)
							}
							if columns >= 80 && status == progressRunning && !strings.Contains(ansi.Strip(frame), "table ") {
								t.Fatalf("table detail missing from %q", frame)
							}
						}
						loaderOptions := LoaderOptions{Action: options.Action, Subject: subject, Detail: options.Detail}
						assertLoaderFrameFits(t, renderRoyalLoaderFrame(loaderOptions, 0, 0), columns)
						assertLoaderFrameFits(t, renderRoyalLoaderResult(loaderOptions, true), columns)
						assertLoaderFrameFits(t, renderRoyalLoaderResult(loaderOptions, false), columns)
					}
				})
			}
		})
	}
}

func TestLoaderLineFallsBackWhenTerminalSizeIsUnavailable(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	originalSize := terminalSize
	t.Cleanup(func() { terminalSize = originalSize })
	terminalSize = func(int) (int, int, error) { return 0, 0, errors.New("no terminal size") }
	frame := renderRoyalProgressFrame(ProgressOptions{
		Action: "importing", Subject: strings.Repeat("dump", 40), Detail: strings.Repeat("table", 40),
	}, 42, progressRunning)
	assertLoaderFrameFits(t, frame, 80)
	if !strings.HasSuffix(frame, "42%") {
		t.Fatalf("percentage missing from fallback frame: %q", frame)
	}
}

func TestLoaderLineKeepsControlCharactersOnOneLine(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalProgressFrame(ProgressOptions{
		Action: "importing\n", Subject: "dump\r\nfile\t.sql", Detail: "table users\nnext",
	}, 42, progressRunning)
	assertLoaderFrameFits(t, frame, 80)
}

func TestRoyalLoadersDoNotWrapWhenClearingWithoutColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "")
	originalStderr, originalSize, originalCheck := os.Stderr, terminalSize, terminalCheck
	t.Cleanup(func() {
		os.Stderr, terminalSize, terminalCheck = originalStderr, originalSize, originalCheck
	})
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	os.Stderr = writer
	terminalSize = func(int) (int, int, error) { return 60, 24, nil }
	terminalCheck = func(*os.File) bool { return true }
	options := ProgressOptions{Action: "importing", Subject: strings.Repeat("dump", 20), Total: 100}
	progress := NewRoyalProgressLoader(options)
	progress.Start()
	progress.SetDetail("table " + strings.Repeat("records", 20))
	progress.Set(42)
	progress.Finish(false)
	if err := RunWithRoyalLoader(LoaderOptions{
		Action: "preparing", Subject: options.Subject, ShowResult: true, Minimum: time.Millisecond,
	}, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.FieldsFunc(string(output), func(r rune) bool { return r == '\r' || r == '\n' }) {
		assertLoaderFrameFits(t, line, 60)
	}
	if strings.Count(string(output), "\n") != 2 {
		t.Fatalf("expected only two final result newlines: %q", output)
	}
}

func assertLoaderFrameFits(t *testing.T, frame string, columns int) {
	t.Helper()
	if width := ansi.StringWidth(frame); width >= columns {
		t.Fatalf("frame width %d would wrap in %d columns: %q", width, columns, frame)
	}
	if strings.ContainsFunc(ansi.Strip(frame), unicode.IsControl) {
		t.Fatalf("frame contains terminal control characters: %q", frame)
	}
}
