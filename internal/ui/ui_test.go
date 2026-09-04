package ui

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFZFThemeArgsUseRoyalNoirPalette(t *testing.T) {
	SetTheme("royal-noir")
	args := strings.Join(FZFThemeArgs("ssh> "), "\n")

	expected := []string{
		"--prompt=ssh> ",
		"bg:#05020a",
		"hl:#d4af37",
		"marker:#7c3aed",
		"prompt:#c4b5fd",
		"border:#3b235c",
	}

	for _, value := range expected {
		if !strings.Contains(args, value) {
			t.Fatalf("FZFThemeArgs missing %q in %s", value, args)
		}
	}
}

func TestThemeRegistryNormalizesNames(t *testing.T) {
	theme, ok := ThemeByName("Tokyo Night")
	if !ok {
		t.Fatal("ThemeByName should find Tokyo Night")
	}
	if theme.Name != "tokyo-night" {
		t.Fatalf("theme name = %q, want tokyo-night", theme.Name)
	}
	if NormalizeThemeName("Catppuccin_Mocha") != "catppuccin-mocha" {
		t.Fatalf("NormalizeThemeName should replace underscores and spaces")
	}
}

func TestThemeNamesIncludeBuiltins(t *testing.T) {
	names := strings.Join(ThemeNames(), ",")
	for _, name := range []string{"royal-noir", "darcula", "tokyo-night", "dracula", "catppuccin-mocha", "nord", "gruvbox-dark", "everforest-dark", "solarized-dark", "one-dark"} {
		if !strings.Contains(names, name) {
			t.Fatalf("ThemeNames missing %q in %s", name, names)
		}
	}
}

func TestSetThemeChangesFZFPalette(t *testing.T) {
	defer SetTheme("royal-noir")

	if !SetTheme("tokyo-night") {
		t.Fatal("SetTheme should accept tokyo-night")
	}
	args := strings.Join(FZFThemeArgs("theme> "), "\n")

	expected := []string{
		"--prompt=theme> ",
		"bg:#1a1b26",
		"hl:#e0af68",
		"marker:#bb9af7",
		"border:#414868",
	}
	for _, value := range expected {
		if !strings.Contains(args, value) {
			t.Fatalf("FZFThemeArgs missing %q in %s", value, args)
		}
	}
}

func TestRenderRoyalLoaderFrame(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalLoaderFrame(LoaderOptions{
		Action:  "connecting",
		Subject: "api",
	}, 2, 1200*time.Millisecond)

	expected := []string{
		"[░░█████░░░░░░░░░░░]",
		"connecting",
		"api",
	}
	for _, value := range expected {
		if !strings.Contains(frame, value) {
			t.Fatalf("loader frame missing %q in %q", value, frame)
		}
	}
	unexpected := []string{"royal-noir", "ssh handoff", "1.2s"}
	for _, value := range unexpected {
		if strings.Contains(frame, value) {
			t.Fatalf("loader frame should not include %q in %q", value, frame)
		}
	}
}

func TestRenderRoyalLoaderResult(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalLoaderResult(LoaderOptions{
		Subject: "workspace-task_600_8048",
	}, true)

	expected := []string{
		"[██████████████████]",
		"done",
		"workspace-task_600_8048",
	}
	for _, value := range expected {
		if !strings.Contains(frame, value) {
			t.Fatalf("loader result missing %q in %q", value, frame)
		}
	}
}

func TestRenderRoyalLoaderResultUsesCustomSuccessAction(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalLoaderResult(LoaderOptions{
		Subject:       "adami",
		SuccessAction: "imported",
	}, true)

	if !strings.Contains(frame, "imported") || strings.Contains(frame, "done") {
		t.Fatalf("loader result should use custom action: %q", frame)
	}
}

func TestRenderRoyalProgressFrame(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalProgressFrame(ProgressOptions{
		Action:  "importing",
		Subject: "adami.sql.gz",
	}, 50, progressRunning)

	expected := []string{
		"[█████████░░░░░░░░░]",
		"importing",
		"adami.sql.gz",
		" 50%",
	}
	for _, value := range expected {
		if !strings.Contains(frame, value) {
			t.Fatalf("progress frame missing %q in %q", value, frame)
		}
	}
}

func TestRenderRoyalProgressFrameCompletes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	frame := renderRoyalProgressFrame(ProgressOptions{
		Action:  "importing",
		Subject: "adami.sql.gz",
	}, 100, progressSuccess)

	expected := []string{
		"[██████████████████]",
		"completed",
		"adami.sql.gz",
		"100%",
	}
	for _, value := range expected {
		if !strings.Contains(frame, value) {
			t.Fatalf("completed progress frame missing %q in %q", value, frame)
		}
	}
}

func TestProgressPercentClamps(t *testing.T) {
	tests := []struct {
		name    string
		current int64
		total   int64
		want    int
	}{
		{name: "zero total", current: 10, total: 0, want: 0},
		{name: "half", current: 50, total: 100, want: 50},
		{name: "over", current: 150, total: 100, want: 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := progressPercent(test.current, test.total); got != test.want {
				t.Fatalf("progressPercent(%d, %d) = %d, want %d", test.current, test.total, got, test.want)
			}
		})
	}
}

func TestRoyalProgressLoaderTracksProgress(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	loader := NewRoyalProgressLoader(ProgressOptions{Action: "importing", Subject: "dump.sql", Total: 200})

	loader.Add(40)
	if got := loader.Percent(); got != 20 {
		t.Fatalf("Percent after Add = %d, want 20", got)
	}
	loader.Set(300)
	if got := loader.Percent(); got != 100 {
		t.Fatalf("Percent after Set = %d, want 100", got)
	}
	loader.Set(-1)
	if got := loader.Percent(); got != 0 {
		t.Fatalf("Percent after negative Set = %d, want 0", got)
	}
	loader.Start()
	loader.Finish(true)
}

func TestRunWithRoyalProgressReturnsFunctionResult(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	called := false
	err := RunWithRoyalProgress(ProgressOptions{Action: "creating", Subject: "workspace", Total: 2}, func(progress *RoyalProgressLoader) error {
		called = true
		progress.Add(1)
		return nil
	})
	if err != nil || !called {
		t.Fatalf("RunWithRoyalProgress success err=%v called=%v", err, called)
	}

	want := errors.New("broken")
	got := RunWithRoyalProgress(ProgressOptions{Action: "creating", Subject: "workspace", Total: 2}, func(progress *RoyalProgressLoader) error {
		progress.Add(1)
		return want
	})
	if !errors.Is(got, want) {
		t.Fatalf("RunWithRoyalProgress error = %v, want %v", got, want)
	}
}

func TestRunWithRoyalLoaderReturnsFunctionResultWhenDisabled(t *testing.T) {
	t.Setenv("DVV_NO_LOADER", "1")
	called := false
	err := RunWithRoyalLoader(LoaderOptions{Action: "testing"}, func() error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Fatalf("RunWithRoyalLoader success err=%v called=%v", err, called)
	}

	want := errors.New("broken")
	got := RunWithRoyalLoader(LoaderOptions{Action: "testing"}, func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Fatalf("RunWithRoyalLoader error = %v, want %v", got, want)
	}
}

func TestRunWithRoyalLoaderDoesNotHangOnPanic(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("DVV_NO_LOADER", "")
	originalTerminalCheck := terminalCheck
	terminalCheck = func(*os.File) bool { return true }
	t.Cleanup(func() {
		terminalCheck = originalTerminalCheck
	})

	defer func() {
		recovered := recover()
		if recovered != "boom" {
			t.Fatalf("panic = %#v, want boom", recovered)
		}
	}()

	_ = RunWithRoyalLoader(LoaderOptions{Action: "testing", ShowResult: true}, func() error {
		panic("boom")
	})
}
