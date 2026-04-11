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

// runVerbose streams all command output directly to stdout/stderr.
// Used for setup commands when --verbose is active.
func (c client) runVerbose(args []string, stdin []byte) *Result {
	ctx := context.Background()
	cancel := func() {}
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, "incus", args...)
	fmt.Fprintf(os.Stderr, terminal.ColorDim+"[verbose] incus %s"+terminal.ColorReset+"\n", strings.Join(args, " "))

	var stdout, stderr bytes.Buffer
	dimOut := &dimWriter{w: os.Stdout}
	dimErr := &dimWriter{w: os.Stderr}
	cmd.Stdout = io.MultiWriter(&stdout, dimOut)
	cmd.Stderr = io.MultiWriter(&stderr, dimErr)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	err := cmd.Run()
	dimOut.flush()
	dimErr.flush()
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

// dimWriter wraps an io.Writer and prints each line surrounded by dim/reset ANSI codes.
type dimWriter struct {
	w   io.Writer
	buf strings.Builder
}

func (d *dimWriter) writeLine(line string) error {
	_, err := fmt.Fprintf(d.w, "%s  > %s%s\n", terminal.ColorDim, line, terminal.ColorReset)
	return err
}

func (d *dimWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			line := d.buf.String()
			d.buf.Reset()
			if err := d.writeLine(line); err != nil {
				return 0, err
			}
		} else {
			d.buf.WriteByte(b)
		}
	}
	return len(p), nil
}

func (d *dimWriter) flush() {
	if d.buf.Len() > 0 {
		_ = d.writeLine(d.buf.String())
		d.buf.Reset()
	}
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

	if c.verbose {
		fmt.Fprintf(os.Stderr, terminal.ColorDim+"[verbose] incus %s"+terminal.ColorReset+"\n", strings.Join(args, " "))
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
		if progress != nil {
			stdoutWriter = io.MultiWriter(&stdout, progress)
			stderrWriter = io.MultiWriter(&stderr, progress)
		}
	} else {
		// Even for quiet commands, show a spinner so the terminal doesn't
		// appear frozen while waiting for the Incus daemon to respond.
		progress = newTerminalSpinnerWriter()
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
