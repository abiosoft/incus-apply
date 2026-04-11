package incus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
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
