package ui

import (
	"os"
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

var terminalSize = term.GetSize

func loaderLine(bar, action, subject, detail, suffix string) string {
	columns, _, err := terminalSize(int(os.Stderr.Fd()))
	if err != nil || columns <= 0 {
		columns = 80
	}
	// Leave the final column unused so terminal autowrap never moves the cursor.
	width := columns - 1
	prefix := "  " + bar + " " + action
	if suffix != "" {
		suffix = " " + suffix
	}
	subject = loaderText(subject)
	detail = loaderText(detail)
	available := width - ansi.StringWidth(prefix) - ansi.StringWidth(suffix)
	if subject != "" {
		available--
	}
	if detail != "" {
		available--
	}
	available = max(0, available)
	subjectWidth := ansi.StringWidth(subject)
	detailWidth := ansi.StringWidth(detail)
	subjectLimit := min(subjectWidth, available)
	if subject != "" && detail != "" {
		subjectLimit = min(subjectWidth, max(available-detailWidth, available/2))
	}
	detailLimit := min(detailWidth, available-subjectLimit)
	parts := []string{prefix}
	if subjectLimit > 0 {
		parts = append(parts, Accent(ansi.Truncate(subject, subjectLimit, "…")))
	}
	if detailLimit > 0 {
		parts = append(parts, Muted(ansi.Truncate(detail, detailLimit, "…")))
	}
	body := ansi.Truncate(strings.Join(parts, " "), max(0, width-ansi.StringWidth(suffix)), "…")
	return ansi.Truncate(body+suffix, width, "")
}

func loaderText(value string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value))
}
