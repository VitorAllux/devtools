package systemconfig

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VitorAllux/devtools/internal/config"
)

type keyActionRunner struct {
	configFakeRunner
	output []byte
	err    error
	calls  int
}

func (r *keyActionRunner) OutputWithInput(context.Context, string, []byte, string, ...string) ([]byte, error) {
	r.calls++
	if r.calls == 1 {
		return r.output, r.err
	}
	return nil, nil
}

func TestAddCustomKeyAcceptsFZFNoMatchAction(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprintf("filtered=%v", filtered), func(t *testing.T) {
			t.Setenv("NO_COLOR", "1")
			t.Setenv("MAC_TEST_VAR", "")
			cfg := &config.Config{ConfigFile: filepath.Join(t.TempDir(), "config.env"), Project: config.DefaultProjectConfig()}
			runner := &keyActionRunner{output: []byte("N\n"), err: exec.Command("sh", "-c", "exit 1").Run()}
			input, inputWriter, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer input.Close()
			defer inputWriter.Close()
			output, outputWriter, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			defer outputWriter.Close()
			originalInput, originalOutput := os.Stdin, os.Stdout
			os.Stdin, os.Stdout = input, outputWriter
			t.Cleanup(func() { os.Stdin, os.Stdout = originalInput, originalOutput })
			done := make(chan string, 1)
			go func() {
				prompts := []string{"Config key: ", "Config value: ", "globally to Bash/Zsh? [y/N]: "}
				replies := []string{"MAC_TEST_VAR\n", "literal $HOME value\n", "n\n"}
				var transcript strings.Builder
				index := 0
				buffer := make([]byte, 1)
				for {
					n, err := output.Read(buffer)
					transcript.Write(buffer[:n])
					if index < len(prompts) && strings.HasSuffix(transcript.String(), prompts[index]) {
						_, _ = inputWriter.WriteString(replies[index])
						index++
					}
					if err != nil {
						done <- transcript.String()
						return
					}
				}
			}()
			entries := []Entry{}
			if filtered {
				entries = []Entry{{Key: "OTHER_KEY", Category: "Custom", Value: "other"}}
			}
			err = (Manager{Config: cfg, Runner: runner}).keyHub(context.Background(), Category{ID: "custom", Label: "Custom"}, entries)
			_ = outputWriter.Close()
			transcript := <-done
			if err != nil {
				t.Fatalf("no-match action returned an error: %v\n%s", err, transcript)
			}
			values, err := config.ReadEnvFile(cfg.ConfigFile)
			if err != nil || values["MAC_TEST_VAR"] != "literal $HOME value" || runner.calls != 2 {
				t.Fatalf("custom variable was not created: values=%v calls=%d err=%v\n%s", values, runner.calls, err, transcript)
			}
		})
	}
}

func TestConfigKeyHubDistinguishesCancellationAndFailure(t *testing.T) {
	for _, code := range []int{1, 2, 130} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			runner := &keyActionRunner{err: exec.Command("sh", "-c", fmt.Sprintf("exit %d", code)).Run()}
			cfg := &config.Config{Project: config.DefaultProjectConfig()}
			err := (Manager{Config: cfg, Runner: runner}).keyHub(context.Background(), Category{ID: "custom", Label: "Custom"}, []Entry{})
			if (err != nil) != (code == 2) {
				t.Fatalf("fzf exit %d returned %v", code, err)
			}
		})
	}
}

func TestEmptyConfigKeyHubShowsActionsWithoutPreview(t *testing.T) {
	cfg := &config.Config{Project: config.DefaultProjectConfig()}
	cfg.Project.System.ConfigHub.Shortcuts.Add = "shift+a"
	args := configFZFArgs(Category{ID: "custom", Label: "Custom"}, cfg.SystemConfigHubKeys(), true)
	joined := strings.Join(args, "\n")
	if strings.Contains(joined, "--preview=") || !strings.Contains(joined, "No custom keys yet.") || !strings.Contains(joined, "Shift+A") || !containsString(args, "--expect=A") {
		t.Fatalf("empty hub should show configured add action without a preview: %v", args)
	}
}
