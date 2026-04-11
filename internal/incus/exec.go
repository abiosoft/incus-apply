package incus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/abiosoft/incus-apply/internal/terminal"
)

// run executes an incus command while showing transient progress in interactive terminals.
func (c client) run(args []string, stdin []byte) *Result {
	return c.execCmd(args, stdin, true)
}

// runQuiet executes an incus command while still capturing all output.
func (c client) runQuiet(args []string, stdin []byte) *Result {
	return c.execCmd(args, stdin, false)
}

func (c client) runWithProgress(args []string, stdin []byte, progressLabel string) *Result {
	return c.execCmd(args, stdin, true, progressLabel)
}

// execCmd is the shared implementation for run and runQuiet.
func (c client) execCmd(args []string, stdin []byte, showProgress bool, progressLabel ...string) *Result {
	ctx := context.Background()
	cancel := func() {}
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, "incus", args...)

	if c.debug {
		fmt.Printf("[debug] %s\n", cmd.String())
	}

	var stdout, stderr bytes.Buffer
	stdoutWriter := io.Writer(&stdout)
	stderrWriter := io.Writer(&stderr)
	var progress *progressWriter
	if showProgress {
		label := ""
		if len(progressLabel) > 0 {
			label = progressLabel[0]
		}
		progress = newTerminalProgressWriter(label)
	}
	if progress != nil {
		stdoutWriter = io.MultiWriter(&stdout, progress)
		stderrWriter = io.MultiWriter(&stderr, progress)
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter

	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	err := cmd.Run()
	if progress != nil {
		progress.Finish()
	}

	result := &Result{
		Command: "incus " + strings.Join(args, " "),
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result.Error = fmt.Errorf("command timed out after %s", c.timeout)
			return result
		}
		result.Error = fmt.Errorf("%w: %s", err, strings.TrimSpace(result.Stderr))
	}
	return result
}

type progressWriter struct {
	mu       sync.Mutex
	line     strings.Builder
	shown    bool
	onStart  func()
	onUpdate func(string)
	onClear  func()
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
	return newProgressWriter(func() {
		terminal.RewriteLine(prefix)
	}, func(text string) {
		terminal.RewriteLine(prefix + text)
	}, terminal.ClearCurrentLine)
}

func setupProgressLabel(current, total int) string {
	if current <= 0 || total <= 0 {
		return "  └─ running setup... "
	}
	return fmt.Sprintf("  └─ running setup %d of %d... ", current, total)
}

func waitForAgentProgressLabel() string {
	return "  └─ waiting for incus agent... "
}

func restartProgressLabel() string {
	return "  └─ restarting... "
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
}
