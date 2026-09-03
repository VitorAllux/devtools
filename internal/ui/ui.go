package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	RoyalLoaderWidth = 18

	reset = "\033[0m"
	bold  = "\033[1m"
	dim   = "\033[2m"
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
	return color(dim+ansiHex(themeColors().Muted), text)
}

func Muted(text string) string {
	return color(ansiHex(themeColors().Muted), text)
}

func Purple(text string) string {
	return color(ansiHex(themeColors().Accent), text)
}

func Accent(text string) string {
	return color(bold+ansiHex(themeColors().AccentSoft), text)
}

func Cyan(text string) string {
	return color(ansiHex(themeColors().AccentSoft), text)
}

func Gold(text string) string {
	return color(ansiHex(themeColors().Status), text)
}

func Success(text string) string {
	return color(bold+ansiHex(themeColors().Success), text)
}

func Danger(text string) string {
	return color(bold+ansiHex(themeColors().Danger), text)
}

func Crown(text string) string {
	return color(bold+ansiHex(themeColors().Status), text)
}

func Badge(text string) string {
	colors := themeColors()
	return color(ansiHex(colors.Muted), "[") + color(bold+ansiHex(colors.Status), text) + color(ansiHex(colors.Muted), "]")
}

func Title(text string) {
	fmt.Printf("\n%s %s\n%s\n\n",
		Crown("dvv"),
		Accent(text),
		Muted(strings.Repeat("-", 48)),
	)
}

func Info(format string, args ...any) {
	fmt.Printf("%s %s %s\n", Crown("dvv"), Purple("::"), fmt.Sprintf(format, args...))
}

func OK(format string, args ...any) {
	fmt.Printf("%s %s %s\n", Crown("dvv"), Gold("ok"), fmt.Sprintf(format, args...))
}

func Warn(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s %s\n", Crown("dvv"), Gold("!"), fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s %s\n", Crown("dvv"), Danger("x"), fmt.Sprintf(format, args...))
}

func Prompt(label string) (string, error) {
	fmt.Printf("%s %s %s: ", Crown("dvv"), Purple("?"), label)
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
	Action        string
	Subject       string
	Detail        string
	Minimum       time.Duration
	ShowResult    bool
	SuccessAction string
	FailureAction string
}

type loaderRunResult struct {
	err       error
	panicData any
}

func RunWithRoyalLoader(options LoaderOptions, fn func() error) error {
	if !LoaderEnabled() {
		return fn()
	}

	done := make(chan loaderRunResult, 1)
	go func() {
		result := loaderRunResult{}
		defer func() {
			if recovered := recover(); recovered != nil {
				result.panicData = recovered
				result.err = fmt.Errorf("loader task panicked: %v", recovered)
			}
			done <- result
		}()
		result.err = fn()
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
		fmt.Fprint(os.Stderr, "\r\033[2K")
		if !useColor() {
			fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", clearWidth)+"\r")
		}
	}
	finish := func(err error) {
		clear()
		if options.ShowResult || err != nil {
			fmt.Fprintln(os.Stderr, renderRoyalLoaderResult(options, err == nil))
		}
	}

	result := loaderRunResult{}
	completed := false
	doneCh := (<-chan loaderRunResult)(done)
	render()

	for {
		select {
		case runResult := <-doneCh:
			result = runResult
			completed = true
			doneCh = nil
			if options.Minimum <= 0 || time.Since(start) >= options.Minimum {
				finish(result.err)
				if result.panicData != nil {
					panic(result.panicData)
				}
				return result.err
			}
		case <-ticker.C:
			render()
			if completed && time.Since(start) >= options.Minimum {
				finish(result.err)
				if result.panicData != nil {
					panic(result.panicData)
				}
				return result.err
			}
		}
	}
}

var terminalCheck = isTerminal

func LoaderEnabled() bool {
	return os.Getenv("DVV_NO_LOADER") != "1" && terminalCheck(os.Stderr)
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

func renderRoyalLoaderResult(options LoaderOptions, ok bool) string {
	action := strings.TrimSpace(options.SuccessAction)
	if action == "" {
		action = "done"
	}
	style := Success
	if !ok {
		action = strings.TrimSpace(options.FailureAction)
		if action == "" {
			action = "failed"
		}
		style = Danger
	}

	parts := []string{
		loaderResultBar(ok),
		style(action),
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
	const segment = 5

	travel := RoyalLoaderWidth - segment
	cycle := travel * 2
	position := index % cycle
	if position > travel {
		position = cycle - position
	}

	var builder strings.Builder
	builder.WriteString(Muted("["))
	for i := 0; i < RoyalLoaderWidth; i++ {
		if i >= position && i < position+segment {
			builder.WriteString(Crown("█"))
			continue
		}
		builder.WriteString(Purple("░"))
	}
	builder.WriteString(Muted("]"))
	return builder.String()
}

func loaderResultBar(ok bool) string {
	fill := Crown
	if ok {
		fill = Success
	} else {
		fill = Danger
	}

	var builder strings.Builder
	builder.WriteString(Muted("["))
	for i := 0; i < RoyalLoaderWidth; i++ {
		builder.WriteString(fill("█"))
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
		"--tabstop=4",
		"--prompt=" + value,
		"--pointer=>>",
		"--marker=+",
		"--color=" + fzfColorSpec(themeColors()),
	}
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
