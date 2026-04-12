package incus

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/resource"
)

func TestExecCmdTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based timeout test is not supported on Windows")
	}

	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "incus")
	script := "#!/bin/sh\nsleep 1\nprintf 'delayed'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	result := client{timeout: 10 * time.Millisecond}.runQuiet([]string{"version"}, nil)
	if result.Error == nil {
		t.Fatal("runQuiet() error = nil, want timeout error")
	}
	if !strings.Contains(result.Error.Error(), "timed out") {
		t.Fatalf("timeout error = %q, want to contain %q", result.Error.Error(), "timed out")
	}
}

func TestProgressWriterShowsLastLineAndClears(t *testing.T) {
	var updates []string
	cleared := 0
	prefix := cloudInitProgressLabel()
	writer := newProgressWriter(nil,
		func(text string) { updates = append(updates, prefix+text) },
		func() { cleared++ },
	)

	if _, err := writer.Write([]byte("Downloading\nInstalling")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	writer.Finish()

	if len(updates) == 0 {
		t.Fatal("updates = 0, want progress updates")
	}
	if updates[len(updates)-1] != prefix+"Installing" {
		t.Fatalf("last update = %q, want %q", updates[len(updates)-1], prefix+"Installing")
	}
	if cleared != 1 {
		t.Fatalf("cleared = %d, want 1", cleared)
	}
}

func TestProgressWriterDisplaysInitialLabel(t *testing.T) {
	var updates []string
	cleared := 0
	writer := newProgressWriter(func() { updates = append(updates, "waiting for incus agent...") },
		func(text string) { updates = append(updates, text) },
		func() { cleared++ },
	)

	if len(updates) != 1 {
		t.Fatalf("updates = %d, want 1", len(updates))
	}
	if updates[0] != "waiting for incus agent..." {
		t.Fatalf("initial update = %q, want %q", updates[0], "waiting for incus agent...")
	}

	writer.Finish()
	if cleared != 1 {
		t.Fatalf("cleared = %d, want 1", cleared)
	}
}

func TestCloudInitProgressLabel(t *testing.T) {
	if got := cloudInitProgressLabel(); got != "  └─ waiting for cloud-init: " {
		t.Fatalf("cloudInitProgressLabel() = %q, want %q", got, "  └─ waiting for cloud-init: ")
	}
}

func TestWaitForAgentProgressLabel(t *testing.T) {
	if got := waitForAgentProgressLabel(); got != "  └─ waiting for incus agent " {
		t.Fatalf("waitForAgentProgressLabel() = %q, want %q", got, "  └─ waiting for incus agent ")
	}
}

func TestWaitInstanceAgentUsesTimeoutAndProject(t *testing.T) {
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "incus")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{Base: config.Base{Type: "instance", Name: "vm1", Project: "prod"}, InstanceFields: config.InstanceFields{VM: true}}
	result := client{timeout: 6 * time.Second}.WaitInstanceAgent(res)
	if result.Error != nil {
		t.Fatalf("WaitInstanceAgent() error = %v", result.Error)
	}
	if !strings.Contains(result.Command, "wait vm1 agent --interval 1 --timeout 6 --project prod") {
		t.Fatalf("command = %q, want wait agent command with timeout and project", result.Command)
	}
}

func TestBuildInstanceCreateArgsEphemeral(t *testing.T) {
	res := &config.Resource{
		Base:           config.Base{Type: "instance", Name: "tmp1"},
		InstanceFields: config.InstanceFields{Image: "images:alpine/3.19", Ephemeral: true},
	}
	meta, _ := resource.GetTypeMeta("instance")
	args, _ := client{}.buildCreateCommand(meta, res)
	cmd := strings.Join(args, " ")
	if !strings.Contains(cmd, "--ephemeral") {
		t.Fatalf("command = %q, want --ephemeral flag", cmd)
	}
}
