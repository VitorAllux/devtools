package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	reset      = "\033[0m"
	bold       = "\033[1m"
	dim        = "\033[2m"
	purple     = "\033[38;2;124;58;237m"
	purpleSoft = "\033[38;2;196;181;253m"
	gold       = "\033[38;2;212;175;55m"
	muted      = "\033[38;2;139;126;163m"
	danger     = "\033[38;2;248;113;113m"
)

func useColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}

func color(code string, text string) string {
	if !useColor() {
		return text
	}
	return code + text + reset
}

func Bold(text string) string {
	return color(bold, text)
}

func Dim(text string) string {
	return color(dim+muted, text)
}

func Muted(text string) string {
	return color(muted, text)
}

func Purple(text string) string {
	return color(purple, text)
}

func Accent(text string) string {
	return color(bold+purpleSoft, text)
}

func Cyan(text string) string {
	return color(purpleSoft, text)
}

func Gold(text string) string {
	return color(gold, text)
}

func Danger(text string) string {
	return color(bold+danger, text)
}

func Crown(text string) string {
	return color(bold+gold, text)
}

func Badge(text string) string {
	return color(muted, "[") + color(bold+gold, text) + color(muted, "]")
}

func Title(text string) {
	fmt.Printf("\n%s %s\n%s\n\n",
		color(bold+gold, "dvv"),
		color(bold+purpleSoft, text),
		color(muted, strings.Repeat("-", 48)),
	)
}

func Info(format string, args ...any) {
	fmt.Printf("%s %s %s\n", color(bold+gold, "dvv"), color(purple, "::"), fmt.Sprintf(format, args...))
}

func OK(format string, args ...any) {
	fmt.Printf("%s %s %s\n", color(bold+gold, "dvv"), color(gold, "ok"), fmt.Sprintf(format, args...))
}

func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s %s\n", color(bold+gold, "dvv"), color(gold, "!"), fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s %s\n", color(bold+gold, "dvv"), color(danger, "x"), fmt.Sprintf(format, args...))
}

func Prompt(label string) (string, error) {
	fmt.Printf("%s %s %s: ", color(bold+gold, "dvv"), color(purple, "?"), label)
	reader := bufio.NewReader(os.Stdin)
	value, err := reader.ReadString('\n')
	if err != nil && len(value) == 0 {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func Confirm(label string) bool {
	value, err := Prompt(label + " [y/N]")
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}

func RunWithLoader(message string, fn func() error) error {
	return RunWithLoaderMin(message, 0, fn)
}

func RunWithLoaderMin(message string, minimum time.Duration, fn func() error) error {
	return RunWithRoyalLoader(LoaderOptions{
		Action:  message,
		Minimum: minimum,
	}, fn)
}

type LoaderOptions struct {
	Action  string
	Subject string
	Detail  string
	Minimum time.Duration
}

func RunWithRoyalLoader(options LoaderOptions, fn func() error) error {
	if !LoaderEnabled() {
		return fn()
	}

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	start := time.Now()
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	index := 0
	clearWidth := 160
	render := func() {
		elapsed := time.Since(start).Truncate(100 * time.Millisecond)
		fmt.Fprintf(os.Stderr, "\r%s", renderRoyalLoaderFrame(options, index, elapsed))
		index++
	}
	clear := func() {
		fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", clearWidth)+"\r")
	}

	var result error
	completed := false
	doneCh := (<-chan error)(done)
	render()

	for {
		select {
		case err := <-doneCh:
			result = err
			completed = true
			doneCh = nil
			if options.Minimum <= 0 || time.Since(start) >= options.Minimum {
				clear()
				return result
			}
		case <-ticker.C:
			render()
			if completed && time.Since(start) >= options.Minimum {
				clear()
				return result
			}
		}
	}
}

func LoaderEnabled() bool {
	return os.Getenv("DVV_NO_LOADER") != "1" && isTerminal(os.Stderr)
}

func renderRoyalLoaderFrame(options LoaderOptions, index int, _ time.Duration) string {
	action := strings.TrimSpace(options.Action)
	if action == "" {
		action = "working"
	}

	parts := []string{
		loaderBar(index),
		Crown(action),
	}
	if subject := strings.TrimSpace(options.Subject); subject != "" {
		parts = append(parts, Accent(subject))
	}
	if detail := strings.TrimSpace(options.Detail); detail != "" {
		parts = append(parts, Muted(detail))
	}
	return "  " + strings.Join(parts, " ")
}

func loaderBar(index int) string {
	const width = 16
	const segment = 5

	travel := width - segment
	cycle := travel * 2
	position := index % cycle
	if position > travel {
		position = cycle - position
	}

	var builder strings.Builder
	builder.WriteString(Muted("["))
	for i := 0; i < width; i++ {
		if i >= position && i < position+segment {
			builder.WriteString(Crown("█"))
			continue
		}
		builder.WriteString(Purple("░"))
	}
	builder.WriteString(Muted("]"))
	return builder.String()
}

func FZFThemeArgs(prompt ...string) []string {
	value := "dvv> "
	if len(prompt) > 0 && strings.TrimSpace(prompt[0]) != "" {
		value = prompt[0]
	}
	return []string{
		"--ansi",
		"--height=~85%",
		"--min-height=18",
		"--layout=reverse",
		"--border=rounded",
		"--margin=1,2",
		"--padding=1,2",
		"--info=inline-right",
		"--separator=-",
		"--cycle",
		"--scroll-off=3",
		"--keep-right",
		"--tabstop=4",
		"--prompt=" + value,
		"--pointer=>>",
		"--marker=+",
		"--color=fg:#f8f5ff,bg:#05020a,hl:#d4af37,fg+:#ffffff,bg+:#170f24,hl+:#e6c76a,pointer:#d4af37,marker:#7c3aed,prompt:#c4b5fd,spinner:#d4af37,header:#c4b5fd,border:#3b235c",
	}
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
