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

func TestCloudInitPowerStateMode(t *testing.T) {
	cases := []struct {
		name   string
		config map[string]string
		want   string
	}{
		{
			name:   "user-data poweroff",
			config: map[string]string{"cloud-init.user-data": "#cloud-config\npower_state:\n  mode: poweroff\n"},
			want:   "poweroff",
		},
		{
			name:   "vendor-data reboot",
			config: map[string]string{"cloud-init.vendor-data": "#cloud-config\npower_state:\n  mode: reboot\n"},
			want:   "reboot",
		},
		{
			name:   "no power_state",
			config: map[string]string{"cloud-init.user-data": "#cloud-config\npackages:\n  - nginx\n"},
			want:   "",
		},
		{
			name:   "empty config",
			config: map[string]string{},
			want:   "",
		},
		{
			name:   "invalid yaml",
			config: map[string]string{"cloud-init.user-data": "{{not yaml}}"},
			want:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
			res.Config = tc.config
			if got := cloudInitPowerStateMode(res); got != tc.want {
				t.Fatalf("cloudInitPowerStateMode() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWaitCloudInitRebootWaitsAndRetries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on Windows")
	}

	// Fake incus binary — happy reboot path:
	//   status --wait: fails (connection severed by reboot)
	//   exec true: succeeds (instance still up before reboot)
	//   test -f boot-finished: succeeds (first boot completed)
	//   waitRunning / list: always running
	//   test -f boot-finished (poll after reboot): succeeds again
	//   cat result.json: no errors
	dir := t.TempDir()
	callFile := filepath.Join(dir, "exec_calls")
	script := `#!/bin/sh
case "$1" in
  exec)
    for arg in "$@"; do
      case "$arg" in
        sh)
          # cloud-init status --wait call
          count=$(cat "` + callFile + `" 2>/dev/null || echo 0)
          echo $((count + 1)) > "` + callFile + `"
          exit 1  # connection severed by reboot
          ;;
        /var/lib/cloud/instance/boot-finished)
          exit 0  # present from first boot (and again after second boot)
          ;;
        /var/lib/cloud/data/result.json)
          printf '{"v1":{"datasource":"DataSourceNoCloud","errors":[]}}'
          exit 0
          ;;
      esac
    done
    exit 0
    ;;
  list) printf "running\n" ;;
esac
`
	scriptPath := filepath.Join(dir, "incus")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
	res.Config = map[string]string{
		"cloud-init.user-data": "#cloud-config\npower_state:\n  mode: reboot\n  message: Rebooting after kernel installation\n",
	}
	result := client{}.WaitCloudInit(res)
	if result.Error != nil {
		t.Fatalf("WaitCloudInit() error = %v, want nil (reboot, result.json shows no errors)", result.Error)
	}
	// status --wait was called only once (before the reboot).
	data, _ := os.ReadFile(callFile)
	if string(data) != "1\n" {
		t.Fatalf("cloud-init status --wait call count = %q, want 1", string(data))
	}
}

func TestWaitCloudInitRebootFallsBackToStatusWait(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on Windows")
	}

	// Fake incus binary — fallback path:
	//   status --wait: 1st call fails (reboot severs exec) → 2nd call succeeds
	//   exec true: fails (instance not reachable, reboot in progress)
	//   test -f boot-finished: always fails (not yet written on new boot)
	//   list: always running
	dir := t.TempDir()
	callFile := filepath.Join(dir, "exec_calls")
	script := `#!/bin/sh
case "$1" in
  exec)
    for arg in "$@"; do
      case "$arg" in
        sh)
          count=$(cat "` + callFile + `" 2>/dev/null || echo 0)
          echo $((count + 1)) > "` + callFile + `"
          if [ "$count" -eq 0 ]; then exit 1; fi  # first call fails (reboot)
          exit 0                                   # second call succeeds (fallback)
          ;;
        /var/lib/cloud/instance/boot-finished) exit 1 ;;  # never found
        /var/lib/cloud/data/result.json)        exit 1 ;;  # never found
        true) exit 1 ;;  # not reachable (in reboot)
      esac
    done
    exit 0
    ;;
  list) printf "running\n" ;;
esac
`
	scriptPath := filepath.Join(dir, "incus")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
	res.Config = map[string]string{
		"cloud-init.user-data": "#cloud-config\npower_state:\n  mode: reboot\n",
	}
	result := client{}.WaitCloudInit(res)
	if result.Error != nil {
		t.Fatalf("WaitCloudInit() error = %v, want nil (fallback status --wait succeeded)", result.Error)
	}
	// status --wait called twice: first (fails/reboot), second (fallback/succeeds).
	data, _ := os.ReadFile(callFile)
	if string(data) != "2\n" {
		t.Fatalf("cloud-init status --wait call count = %q, want 2", string(data))
	}
}

func TestWaitCloudInitRebootAbortsWhenInstanceStops(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on Windows")
	}

	// Fake incus binary — abort path:
	//   all exec calls fail (instance not reachable, went down and never came back)
	//   list: always empty (instance stopped)
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
  exec) exit 1 ;;
  list) printf "" ;;
esac
`
	scriptPath := filepath.Join(dir, "incus")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
	res.Config = map[string]string{
		"cloud-init.user-data": "#cloud-config\npower_state:\n  mode: reboot\n",
	}
	result := client{}.WaitCloudInit(res)
	if result.Error == nil {
		t.Fatal("WaitCloudInit() error = nil, want error (instance stopped after reboot)")
	}
}

func TestWaitCloudInitSuppressesErrorWhenInstanceStopped(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on Windows")
	}

	// Fake incus binary: exec fails (simulates power_state shutdown),
	// but list reports the instance as stopped (not "running").
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
  exec) exit 1 ;;
  list) printf "" ;;  # empty output → not running
esac
`
	scriptPath := filepath.Join(dir, "incus")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{
		Base:           config.Base{Type: "instance", Name: "web"},
		InstanceFields: config.InstanceFields{},
	}
	res.Config = map[string]string{
		"cloud-init.user-data": "#cloud-config\npower_state:\n  mode: poweroff\n",
	}
	result := client{}.WaitCloudInit(res)
	if result.Error != nil {
		t.Fatalf("WaitCloudInit() error = %v, want nil (power_state set, instance stopped)", result.Error)
	}
}

func TestWaitCloudInitPreservesErrorWhenInstanceStillRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on Windows")
	}

	// Fake incus binary: exec fails AND instance is still running → real failure.
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
  exec) exit 1 ;;
  list) printf "running\n" ;;
esac
`
	scriptPath := filepath.Join(dir, "incus")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
	res.Config = map[string]string{
		"cloud-init.user-data": "#cloud-config\npower_state:\n  mode: poweroff\n",
	}
	result := client{}.WaitCloudInit(res)
	if result.Error == nil {
		t.Fatal("WaitCloudInit() error = nil, want error (instance still running)")
	}
}

func TestWaitCloudInitPreservesErrorWhenNoPowerState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on Windows")
	}

	// Fake incus binary: exec fails, instance is stopped, but no power_state
	// declared → treat as real cloud-init failure.
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
  exec) exit 1 ;;
  list) printf "" ;;
esac
`
	scriptPath := filepath.Join(dir, "incus")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	res := &config.Resource{Base: config.Base{Type: "instance", Name: "web"}}
	res.Config = map[string]string{
		"cloud-init.user-data": "#cloud-config\npackages:\n  - nginx\n",
	}
	result := client{}.WaitCloudInit(res)
	if result.Error == nil {
		t.Fatal("WaitCloudInit() error = nil, want error (no power_state, stopped unexpectedly)")
	}
}
