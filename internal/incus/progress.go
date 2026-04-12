package incus

import (
	"os"
	"strings"
	"sync"
	"time"

	"github.com/abiosoft/incus-apply/internal/terminal"
)

type progressWriter struct {
	mu        sync.Mutex
	line      strings.Builder
	shown     bool
	hasOutput bool // true once real command output has been displayed
	onStart   func()
	onUpdate  func(string)
	onClear   func()
	stopSpin  chan struct{}
	spinDone  chan struct{}
}

var spinnerFrames = []string{" ⠋ ", " ⠙ ", " ⠹ ", " ⠸ ", " ⠼ ", " ⠴ ", " ⠦ ", " ⠧ ", " ⠇ ", " ⠏ "}

// startSpinner starts a goroutine that animates a spinner on the current
// terminal line (using prefix as context) while no command output is flowing.
func (w *progressWriter) startSpinner(prefix string) {
	w.stopSpin = make(chan struct{})
	w.spinDone = make(chan struct{})
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer close(w.spinDone)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-w.stopSpin:
				return
			case <-ticker.C:
				w.mu.Lock()
				// Only animate when no real command output has been shown yet.
				// Once output arrives, the spinner stops permanently.
				if w.line.Len() == 0 && !w.hasOutput {
					terminal.RewriteLine(prefix + spinnerFrames[i%len(spinnerFrames)])
					w.shown = true
					i++
				}
				w.mu.Unlock()
			}
		}
	}()
}

func newProgressWriter(onStart func(), onUpdate func(string), onClear func()) *progressWriter {
	w := &progressWriter{onStart: onStart, onUpdate: onUpdate, onClear: onClear}
	if onStart != nil {
		onStart()
		w.shown = true
	}
	return w
}

func newTerminalProgressWriter(prefix string) *progressWriter {
	if !terminal.IsTerminal(os.Stdout) {
		return nil
	}
	w := newProgressWriter(func() {
		terminal.RewriteLine(prefix)
	}, func(text string) {
		terminal.RewriteLine(prefix + text)
	}, terminal.ClearCurrentLine)
	w.startSpinner(prefix)
	return w
}

// newTerminalSpinnerWriter creates a progress writer that shows only a spinner
// (no command output). Used by runQuiet to indicate activity without displaying
// the raw incus command output.
func newTerminalSpinnerWriter() *progressWriter {
	if !terminal.IsTerminal(os.Stdout) {
		return nil
	}
	w := &progressWriter{onClear: terminal.ClearCurrentLine}
	w.startSpinner("")
	return w
}

func waitForAgentProgressLabel() string {
	return "  └─ waiting for incus agent "
}

func cloudInitProgressLabel() string {
	return "  └─ waiting for cloud-init: "
}

func (w *progressWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, b := range p {
		switch b {
		case '\r', '\n':
			w.flushLocked()
		default:
			w.line.WriteByte(b)
		}
	}
	if w.line.Len() > 0 {
		w.updateLocked(w.line.String())
	}
	return len(p), nil
}

func (w *progressWriter) Finish() {
	if w.stopSpin != nil {
		close(w.stopSpin)
		<-w.spinDone
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	w.line.Reset()
	if w.shown && w.onClear != nil {
		w.onClear()
		w.shown = false
	}
}

func (w *progressWriter) flushLocked() {
	if w.line.Len() == 0 {
		return
	}
	w.updateLocked(w.line.String())
	w.line.Reset()
}

func (w *progressWriter) updateLocked(text string) {
	if text == "" || w.onUpdate == nil {
		return
	}
	w.onUpdate(text)
	w.shown = true
	w.hasOutput = true
}
