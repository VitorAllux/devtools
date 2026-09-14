package desktop

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/VitorAllux/devtools/internal/run"
)

func CopyToClipboard(ctx context.Context, runner run.Runner, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("clipboard text cannot be empty")
	}
	for _, candidate := range clipboardCandidates() {
		if _, err := runner.LookPath(candidate.name); err != nil {
			continue
		}
		_, err := runner.OutputWithInput(ctx, "", []byte(text), candidate.name, candidate.args...)
		return err
	}
	return fmt.Errorf("no supported clipboard command found")
}

func OpenURL(ctx context.Context, runner run.Runner, url string) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("URL cannot be empty")
	}
	for _, candidate := range openURLCandidates(url) {
		if _, err := runner.LookPath(candidate.name); err != nil {
			continue
		}
		return runner.Start(ctx, "", candidate.name, candidate.args...)
	}
	return fmt.Errorf("no supported browser opener found")
}

type commandCandidate struct {
	name string
	args []string
}

func clipboardCandidates() []commandCandidate {
	if runtime.GOOS == "darwin" {
		return []commandCandidate{{name: "pbcopy"}}
	}
	return []commandCandidate{
		{name: "wl-copy"},
		{name: "xclip", args: []string{"-selection", "clipboard"}},
		{name: "xsel", args: []string{"--clipboard", "--input"}},
		{name: "clip.exe"},
	}
}

func openURLCandidates(url string) []commandCandidate {
	if runtime.GOOS == "darwin" {
		return []commandCandidate{{name: "open", args: []string{url}}}
	}
	return []commandCandidate{
		{name: "wslview", args: []string{url}},
		{name: "xdg-open", args: []string{url}},
		{name: "gio", args: []string{"open", url}},
	}
}
