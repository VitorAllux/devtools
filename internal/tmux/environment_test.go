package tmux

import (
	"strings"
	"testing"
)

func TestTmuxPreviewPreservesCommandArgs(t *testing.T) {
	preview := tmuxPreviewCommand(tmuxHubShortcuts())

	if strings.Contains(preview, "set -- $display") {
		t.Fatalf("preview should not replace shortcut args with display columns: %s", preview)
	}
	if strings.Contains(preview, "DVV_FZF_COMMANDS") {
		t.Fatalf("preview should render shortcut commands directly: %s", preview)
	}
	for _, want := range []string{"Alt+U", "start/open", "Alt+D", "stop", "Alt+A", "restart API", "Alt+W", "restart Web"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q: %s", want, preview)
		}
	}
}
