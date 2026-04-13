package incus

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/abiosoft/incus-apply/internal/config"
	"gopkg.in/yaml.v3"
)

// --- Embedded scripts ---

//go:embed scripts/cloud-init-wait.sh
var cloudInitWaitScript string

// --- File paths used inside instances ---

const (
	cloudInitBootFinished = "/var/lib/cloud/instance/boot-finished"
	cloudInitResultJSON   = "/var/lib/cloud/data/result.json"
)

func (c client) WaitCloudInit(res *config.Resource) *Result {
	result := c.runCloudInitWait(res)

	mode := cloudInitPowerStateMode(res)
	switch mode {
	case "reboot":
		// The exec connection may be severed when the reboot kicks in.
		// Regardless of the first result, wait for the instance to come back
		// and verify cloud-init completion via boot-finished + result.json.
		return c.waitCloudInitAfterReboot(res)
	case "":
		return result
	default:
		// poweroff (or any other mode): a stopped instance is the intended outcome.
		if result.Error != nil && !c.Running(res) {
			result.Error = nil
		}
		return result
	}
}

// waitCloudInitAfterReboot confirms cloud-init completion after a reboot
// triggered by cloud-init power_state.
//
// Flow:
//  1. Wait up to 10 s for the instance to be running again.
//  2. For VMs, wait for the instance agent.
//  3. Poll for /var/lib/cloud/instance/boot-finished (written when cloud-init
//     finishes the new boot); once present, check result.json for errors.
//  4. If boot-finished never appears, fall back to cloud-init status --wait.
func (c client) waitCloudInitAfterReboot(res *config.Resource) *Result {
	// Wait for the instance to return after the reboot (up to 10 s).
	if err := c.waitRunning(res, 10*time.Second); err != nil {
		return &Result{Error: fmt.Errorf("waiting for instance after reboot: %w", err)}
	}

	// VMs additionally need the instance agent to be ready.
	if res.VM {
		if r := c.WaitInstanceAgent(res); r.Error != nil {
			return r
		}
	}

	// Poll for boot-finished — cloud-init removes and rewrites it on each boot.
	// Once it appears, read result.json to confirm there are no errors.
	const (
		pollInterval = 1 * time.Second
		maxPolls     = 3
	)
	for range maxPolls {
		if c.fileExistsInInstance(res, cloudInitBootFinished) {
			return c.checkCloudInitResult(res)
		}
		time.Sleep(pollInterval)
	}

	// boot-finished not yet written — cloud-init may still be running.
	// Fall back to cloud-init status --wait one more time.
	return c.runCloudInitWait(res)
}

// waitRunning polls until the instance enters the RUNNING state or the timeout
// elapses. A background goroutine checks the instance status up to three times
// over the window; if it detects the instance is not running it signals an
// early abort (the intended reboot became a poweroff, or never happened).
func (c client) waitRunning(res *config.Resource, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	stopped := make(chan struct{}, 1)
	quit := make(chan struct{})
	defer close(quit)

	go func() {
		interval := max(timeout/4, 500*time.Millisecond)
		for range 3 {
			select {
			case <-quit:
				return
			case <-time.After(interval):
			}
			if !c.Running(res) {
				select {
				case stopped <- struct{}{}:
				default:
				}
				return
			}
		}
	}()

	for {
		if c.Running(res) {
			return nil
		}
		select {
		case <-stopped:
			return fmt.Errorf("instance stopped unexpectedly after reboot")
		default:
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for instance to start after reboot")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// fileExistsInInstance checks whether a file exists inside the instance.
func (c client) fileExistsInInstance(res *config.Resource, path string) bool {
	return c.runQuiet(c.instanceExecArgs(res, "test", "-f", path), nil).Error == nil
}

// checkCloudInitResult reads result.json and returns nil error when cloud-init
// reports no errors, or an error listing the failures otherwise.
func (c client) checkCloudInitResult(res *config.Resource) *Result {
	r, found, err := c.readCloudInitResult(res)
	if err != nil {
		return &Result{Error: err}
	}
	if !found {
		return &Result{Error: fmt.Errorf("cloud-init result.json not found after boot-finished")}
	}
	if len(r.V1.Errors) > 0 {
		return &Result{Error: fmt.Errorf("cloud-init errors: %s", strings.Join(r.V1.Errors, "; "))}
	}
	return &Result{}
}

// cloudInitResultData is the structure of /var/lib/cloud/data/result.json.
type cloudInitResultData struct {
	V1 struct {
		Errors []string `json:"errors"`
	} `json:"v1"`
}

// readCloudInitResult reads /var/lib/cloud/data/result.json from inside the
// instance. Returns (result, true, nil) on success, (nil, false, nil) when the
// file is not accessible, or (nil, false, err) on a parse failure.
func (c client) readCloudInitResult(res *config.Resource) (*cloudInitResultData, bool, error) {
	result := c.runQuiet(c.instanceExecArgs(res, "cat", cloudInitResultJSON), nil)
	if result.Error != nil {
		return nil, false, nil // not accessible yet
	}
	var r cloudInitResultData
	if err := json.Unmarshal([]byte(result.Stdout), &r); err != nil {
		return nil, false, fmt.Errorf("parsing cloud-init result.json: %w", err)
	}
	return &r, true, nil
}

// runCloudInitWait runs `cloud-init status --wait` inside the instance.
// Only the tailed cloud-init output log is forwarded to the progress writer;
// the cloud-init status command output itself is suppressed.
func (c client) runCloudInitWait(res *config.Resource) *Result {
	args := c.instanceExecArgs(res, "sh", "-c", cloudInitWaitScript)
	if c.verbose {
		return c.runVerbose(args, nil)
	}
	return c.runWithProgress(args, nil, cloudInitProgressLabel())
}

// cloudInitPowerStateMode returns the power_state.mode value from the
// cloud-init user-data or vendor-data config, or "" if not declared.
func cloudInitPowerStateMode(res *config.Resource) string {
	for _, key := range []string{"cloud-init.user-data", "cloud-init.vendor-data"} {
		data, ok := res.Config[key]
		if !ok {
			continue
		}
		var doc map[string]any
		if err := yaml.Unmarshal([]byte(data), &doc); err != nil {
			continue
		}
		ps, ok := doc["power_state"].(map[string]any)
		if !ok {
			continue
		}
		if mode, _ := ps["mode"].(string); mode != "" {
			return strings.ToLower(mode)
		}
	}
	return ""
}
