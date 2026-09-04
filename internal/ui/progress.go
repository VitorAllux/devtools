package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ProgressOptions struct {
	Action        string
	Subject       string
	Detail        string
	Total         int64
	SuccessAction string
	FailureAction string
}

type RoyalProgressLoader struct {
	options   ProgressOptions
	current   atomic.Int64
	done      chan struct{}
	stopped   chan struct{}
	started   atomic.Bool
	finish    sync.Once
	interval  time.Duration
	clearSize int
}

func NewRoyalProgressLoader(options ProgressOptions) *RoyalProgressLoader {
	return &RoyalProgressLoader{
		options:   options,
		done:      make(chan struct{}),
		stopped:   make(chan struct{}),
		interval:  100 * time.Millisecond,
		clearSize: 160,
	}
}

func RunWithRoyalProgress(options ProgressOptions, fn func(*RoyalProgressLoader) error) (err error) {
	progress := NewRoyalProgressLoader(options)
	progress.Start()
	ok := false
	defer func() {
		if recovered := recover(); recovered != nil {
			progress.Finish(false)
			panic(recovered)
		}
		progress.Finish(ok)
	}()

	err = fn(progress)
	ok = err == nil
	return err
}

func (p *RoyalProgressLoader) Add(n int) {
	if n > 0 {
		p.current.Add(int64(n))
	}
}

func (p *RoyalProgressLoader) Set(current int64) {
	if current < 0 {
		current = 0
	}
	p.current.Store(current)
}

func (p *RoyalProgressLoader) Start() {
	if !LoaderEnabled() || p.options.Total <= 0 {
		return
	}
	if !p.started.CompareAndSwap(false, true) {
		return
	}
	go func() {
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()
		defer close(p.stopped)

		p.render(progressRunning)
		for {
			select {
			case <-ticker.C:
				p.render(progressRunning)
			case <-p.done:
				return
			}
		}
	}()
}

func (p *RoyalProgressLoader) Finish(ok bool) {
	if !p.started.Load() {
		return
	}
	p.finish.Do(func() {
		close(p.done)
		<-p.stopped
		p.clear()
		if ok {
			p.render(progressSuccess)
		} else {
			p.render(progressFailure)
		}
		fmt.Fprint(os.Stderr, "\n")
	})
}

func (p *RoyalProgressLoader) Percent() int {
	return progressPercent(p.current.Load(), p.options.Total)
}

func (p *RoyalProgressLoader) render(status progressStatus) {
	percent := p.Percent()
	if status == progressSuccess {
		percent = 100
	}
	fmt.Fprintf(os.Stderr, "\r%s", renderRoyalProgressFrame(p.options, percent, status))
}

func (p *RoyalProgressLoader) clear() {
	fmt.Fprint(os.Stderr, "\r\033[2K")
	if !useColor() {
		fmt.Fprint(os.Stderr, "\r"+strings.Repeat(" ", p.clearSize)+"\r")
	}
}

type progressStatus int

const (
	progressRunning progressStatus = iota
	progressSuccess
	progressFailure
)

func renderRoyalProgressFrame(options ProgressOptions, percent int, status progressStatus) string {
	percent = clampProgressPercent(percent)
	action := progressActionLabel(options, status)

	parts := []string{
		progressBar(percent, status),
		action,
	}
	if subject := strings.TrimSpace(options.Subject); subject != "" {
		parts = append(parts, Accent(subject))
	}
	if detail := strings.TrimSpace(options.Detail); detail != "" {
		parts = append(parts, Muted(detail))
	}
	parts = append(parts, Muted(fmt.Sprintf("%3d%%", percent)))

	return "  " + strings.Join(parts, " ")
}

func progressActionLabel(options ProgressOptions, status progressStatus) string {
	switch status {
	case progressSuccess:
		action := strings.TrimSpace(options.SuccessAction)
		if action == "" {
			action = "completed"
		}
		return Success(action)
	case progressFailure:
		action := strings.TrimSpace(options.FailureAction)
		if action == "" {
			action = "failed"
		}
		return Danger(action)
	default:
		action := strings.TrimSpace(options.Action)
		if action == "" {
			action = "working"
		}
		return Crown(action)
	}
}

func progressBar(percent int, status progressStatus) string {
	percent = clampProgressPercent(percent)

	filled := (percent * RoyalLoaderWidth) / 100
	fill := Crown
	if status == progressSuccess {
		fill = Success
	}
	if status == progressFailure {
		fill = Danger
	}

	var builder strings.Builder
	builder.WriteString(Muted("["))
	for index := 0; index < RoyalLoaderWidth; index++ {
		if index < filled {
			builder.WriteString(fill("█"))
			continue
		}
		builder.WriteString(Purple("░"))
	}
	builder.WriteString(Muted("]"))
	return builder.String()
}

func progressPercent(current int64, total int64) int {
	if total <= 0 || current <= 0 {
		return 0
	}
	return clampProgressPercent(int((current * 100) / total))
}

func clampProgressPercent(percent int) int {
	if percent < 0 {
		return 0
	}
	if percent > 100 {
		return 100
	}
	return percent
}
