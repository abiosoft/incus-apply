package apply

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abiosoft/incus-apply/internal/config"
	"github.com/abiosoft/incus-apply/internal/incus"
)

type fakeClient struct {
	exists      map[string]bool
	existsErr   map[string]error
	current     map[string]string
	merged      map[string]string
	running     map[string]bool
	createErr   map[string]error
	updateErr   map[string]error
	deleteErr   map[string]error
	startErr    map[string]error
	stopErr     map[string]error
	setupErr    map[string]error
	waitErr     map[string]error
	createCalls []string
	deleteCalls []string
	startCalls  []string
	stopCalls   []string
	waitCalls   []string
	updateCalls []string
	setupCalls  []string
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		exists:    map[string]bool{},
		existsErr: map[string]error{},
		current:   map[string]string{},
		merged:    map[string]string{},
		running:   map[string]bool{},
		createErr: map[string]error{},
		updateErr: map[string]error{},
		deleteErr: map[string]error{},
		startErr:  map[string]error{},
		stopErr:   map[string]error{},
		setupErr:  map[string]error{},
		waitErr:   map[string]error{},
	}
}

func (c *fakeClient) Ping() error { return nil }

func (c *fakeClient) Create(res *config.Resource) *incus.Result {
	key := formatResourceID(res)
	c.createCalls = append(c.createCalls, key)
	return &incus.Result{Error: c.createErr[key]}
}

func (c *fakeClient) Update(res *config.Resource) *incus.Result {
	key := formatResourceID(res)
	c.updateCalls = append(c.updateCalls, key)
	return &incus.Result{Error: c.updateErr[key]}
}

func (c *fakeClient) Delete(res *config.Resource) *incus.Result {
	key := formatResourceID(res)
	c.deleteCalls = append(c.deleteCalls, key)
	return &incus.Result{Error: c.deleteErr[key]}
}

func (c *fakeClient) Exists(res *config.Resource) (bool, error) {
	key := formatResourceID(res)
	if err := c.existsErr[key]; err != nil {
		return false, err
	}
	return c.exists[key], nil
}

func (c *fakeClient) CurrentConfig(res *config.Resource) (string, error) {
	return c.current[formatResourceID(res)], nil
}

func (c *fakeClient) MergedConfig(res *config.Resource) (string, error) {
	return c.merged[formatResourceID(res)], nil
}

func (c *fakeClient) Start(res *config.Resource) *incus.Result {
	key := formatResourceID(res)
	c.startCalls = append(c.startCalls, key)
	return &incus.Result{Error: c.startErr[key]}
}

func (c *fakeClient) Stop(res *config.Resource) *incus.Result {
	key := formatResourceID(res)
	c.stopCalls = append(c.stopCalls, key)
	return &incus.Result{Error: c.stopErr[key]}
}

func (c *fakeClient) Running(res *config.Resource) bool {
	return c.running[formatResourceID(res)]
}

func (c *fakeClient) WaitInstanceAgent(res *config.Resource) *incus.Result {
	key := formatResourceID(res)
	c.waitCalls = append(c.waitCalls, key)
	return &incus.Result{Error: c.waitErr[key]}
}

func (c *fakeClient) RunSetupAction(res *config.Resource, action config.SetupAction, current, total int) *incus.Result {
	key := formatResourceID(res)
	call := key + ":" + string(action.Action) + ":" + string(action.When)
	c.setupCalls = append(c.setupCalls, call)
	err := c.setupErr[call+":"+action.Script]
	if err == nil {
		err = c.setupErr[call]
	}
	return &incus.Result{Error: err}
}

type captureRenderer struct {
	outputs []Output
}

func (r *captureRenderer) Render(output Output) error {
	r.outputs = append(r.outputs, output)
	return nil
}

func writeConfigFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func TestExecutorUpsertCreatesAndStartsInstance(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\n")

	client := newFakeClient()
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Launch: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if len(client.createCalls) != 1 || client.createCalls[0] != "default:instance/web" {
		t.Fatalf("create calls = %v, want [default:instance/web]", client.createCalls)
	}
	if len(client.startCalls) != 1 || client.startCalls[0] != "default:instance/web" {
		t.Fatalf("start calls = %v, want [default:instance/web]", client.startCalls)
	}
	if len(renderer.outputs) != 1 {
		t.Fatalf("renderer outputs = %d, want 1", len(renderer.outputs))
	}
	if got := renderer.outputs[0].Summary; got != "Summary: 1 to create." {
		t.Fatalf("summary = %q, want %q", got, "Summary: 1 to create.")
	}
	if got := renderer.outputs[0].Groups[0].Items[0].Note; got != "launch" {
		t.Fatalf("note = %q, want %q", got, "launch")
	}
}

func TestExecutorUpsertCreateRunsSetupActions(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\nsetup:\n  - action: exec\n    when: create\n    script: echo create\n  - action: file_push\n    when: update\n    path: /etc/app.conf\n    content: hi\n  - action: exec\n    when: always\n    script: echo always\n  - action: exec\n    when: always\n    skip: true\n    script: echo skip\n")

	client := newFakeClient()
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if len(client.createCalls) != 1 || client.createCalls[0] != "default:instance/web" {
		t.Fatalf("create calls = %v, want [default:instance/web]", client.createCalls)
	}
	if len(client.startCalls) != 1 || client.startCalls[0] != "default:instance/web" {
		t.Fatalf("start calls = %v, want [default:instance/web]", client.startCalls)
	}
	if len(client.stopCalls) != 1 || client.stopCalls[0] != "default:instance/web" {
		t.Fatalf("stop calls = %v, want [default:instance/web]", client.stopCalls)
	}
	wantSetup := []string{
		"default:instance/web:exec:create",
		"default:instance/web:file_push:update",
		"default:instance/web:exec:always",
	}
	if strings.Join(client.setupCalls, ",") != strings.Join(wantSetup, ",") {
		t.Fatalf("setup calls = %v, want %v", client.setupCalls, wantSetup)
	}
	if got := renderer.outputs[0].Groups[0].Items[0].Note; got != "setup" {
		t.Fatalf("note = %q, want %q", got, "setup")
	}
	if len(client.startCalls) != 1 {
		t.Fatalf("unexpected launch start calls = %v", client.startCalls)
	}
}

func TestExecutorUpsertAlwaysSetupRunsWithoutConfigUpdate(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\nsetup:\n  - action: exec\n    when: always\n    script: echo always\n")

	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  user.incus-apply.created: \"true\"\n  user.incus-apply.current: '{\"image\":\"images:alpine/3.19\",\"setup\":[{\"action\":\"exec\",\"when\":\"always\",\"script\":\"hash: 9e4ad387b7ad3a5d1a10fb6211\"}]}'\n"
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if len(client.updateCalls) != 0 {
		t.Fatalf("update calls = %v, want none", client.updateCalls)
	}
	if len(client.startCalls) != 1 || len(client.stopCalls) != 1 {
		t.Fatalf("start/stop calls = %v/%v, want one temporary cycle", client.startCalls, client.stopCalls)
	}
	if len(client.setupCalls) != 1 || client.setupCalls[0] != "default:instance/web:exec:always" {
		t.Fatalf("setup calls = %v, want one always exec", client.setupCalls)
	}
	if got := renderer.outputs[0].Summary; got != "Summary: 1 to update." {
		t.Fatalf("summary = %q, want %q", got, "Summary: 1 to update.")
	}
}

func TestExecutorUpsertOptionalSetupFailureContinues(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\nsetup:\n  - action: exec\n    when: create\n    required: false\n    script: echo optional\n  - action: exec\n    when: create\n    script: echo required\n")

	client := newFakeClient()
	client.setupErr["default:instance/web:exec:create:echo optional"] = errors.New("optional failed")
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if len(client.setupCalls) != 2 {
		t.Fatalf("setup calls = %v, want both setup actions to run", client.setupCalls)
	}
	if got := renderer.outputs[0].Summary; got != "Summary: 1 to create." {
		t.Fatalf("summary = %q, want %q", got, "Summary: 1 to create.")
	}
}

func TestExecutorUpsertVMWaitsForAgentBeforeSetup(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: vm1\nvm: true\nimage: images:alpine/3.19\nsetup:\n  - action: exec\n    when: create\n    script: echo create\n")

	client := newFakeClient()
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if len(client.waitCalls) != 1 || client.waitCalls[0] != "default:instance/vm1" {
		t.Fatalf("wait calls = %v, want one VM agent wait", client.waitCalls)
	}
	if len(client.setupCalls) != 1 {
		t.Fatalf("setup calls = %v, want setup to run after wait", client.setupCalls)
	}
}

func TestExecutorUpsertPlanningErrorPreventsApply(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\n")

	client := newFakeClient()
	client.existsErr["default:instance/web"] = errors.New("boom")
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Quiet: true}, client, renderer)

	err := executor.Upsert()
	if err == nil {
		t.Fatal("Upsert() error = nil, want non-nil")
	}
	if len(client.createCalls) != 0 {
		t.Fatalf("create calls = %v, want none", client.createCalls)
	}
	if len(renderer.outputs) != 1 {
		t.Fatalf("renderer outputs = %d, want 1", len(renderer.outputs))
	}
	if got := renderer.outputs[0].Summary; got != "Summary: 1 errors." {
		t.Fatalf("summary = %q, want %q", got, "Summary: 1 errors.")
	}
}

func TestExecutorDeleteRemovesExistingResource(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\n")

	client := newFakeClient()
	client.exists["default:instance/web"] = true
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Delete: true, Yes: true, Quiet: true}, client, renderer)

	if err := executor.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if len(client.deleteCalls) != 1 || client.deleteCalls[0] != "default:instance/web" {
		t.Fatalf("delete calls = %v, want [default:instance/web]", client.deleteCalls)
	}
	if len(renderer.outputs) != 1 {
		t.Fatalf("renderer outputs = %d, want 1", len(renderer.outputs))
	}
	if got := renderer.outputs[0].Summary; got != "Summary: 1 to delete." {
		t.Fatalf("summary = %q, want %q", got, "Summary: 1 to delete.")
	}
}

func TestComputeUpsertDiff_UnmanagedResourceIsMarked(t *testing.T) {
	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  user.key: value\n"

	res := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "web",
			Config: map[string]string{
				"user.key": "updated",
			},
		},
	}

	output, preview, plans := computeUpsertDiff(&Options{}, client, []*config.Resource{res})
	if preview.updated != 1 {
		t.Fatalf("updated count = %d, want 1", preview.updated)
	}
	if len(plans) != 1 || plans[0].action != upsertUpdate {
		t.Fatalf("plans = %#v, want one update plan", plans)
	}
	if len(output.Groups) != 1 || len(output.Groups[0].Items) != 1 {
		t.Fatalf("unexpected output groups: %#v", output.Groups)
	}
	if got := output.Groups[0].Items[0].Note; got != "unmanaged" {
		t.Fatalf("note = %q, want %q", got, "unmanaged")
	}
}

func TestComputeUpsertDiff_RedactsInstanceEnvironmentPreviewValues(t *testing.T) {
	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  environment.DB_PASSWORD: old-secret\n  user.key: value\n"

	res := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "web",
			Config: map[string]string{
				"environment.DB_PASSWORD": "new-secret",
				"user.key":                "updated",
			},
		},
		PreviewRedactPrefixes: []string{"config.environment."},
	}

	output, preview, plans := computeUpsertDiff(&Options{}, client, []*config.Resource{res})
	if preview.updated != 1 {
		t.Fatalf("updated count = %d, want 1", preview.updated)
	}
	if len(plans) != 1 || plans[0].action != upsertUpdate {
		t.Fatalf("plans = %#v, want one update plan", plans)
	}
	changes := output.Groups[0].Items[0].Changes
	if len(changes) != 2 {
		t.Fatalf("changes = %#v, want 2 changes", changes)
	}
	for _, change := range changes {
		switch change.Path {
		case "config.environment.DB_PASSWORD":
			if change.Old != "[redacted]" || change.New != "[redacted]" {
				t.Fatalf("redacted change = %#v, want old/new redacted", change)
			}
		case "config.user.key":
			if change.Old != "value" || change.New != "updated" {
				t.Fatalf("non-redacted change = %#v, want visible values", change)
			}
		default:
			t.Fatalf("unexpected change path = %q", change.Path)
		}
	}
}

func TestComputeUpsertDiff_ShowEnvSkipsPreviewRedaction(t *testing.T) {
	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  environment.DB_PASSWORD: old-secret\n"

	res := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "web",
			Config: map[string]string{
				"environment.DB_PASSWORD": "new-secret",
			},
		},
		PreviewRedactPrefixes: []string{"config.environment."},
	}

	output, preview, plans := computeUpsertDiff(&Options{ShowEnv: true}, client, []*config.Resource{res})
	if preview.updated != 1 {
		t.Fatalf("updated count = %d, want 1", preview.updated)
	}
	if len(plans) != 1 || plans[0].action != upsertUpdate {
		t.Fatalf("plans = %#v, want one update plan", plans)
	}
	changes := output.Groups[0].Items[0].Changes
	if len(changes) != 1 {
		t.Fatalf("changes = %#v, want 1 change", changes)
	}
	if changes[0].Old != "old-secret" || changes[0].New != "new-secret" {
		t.Fatalf("change = %#v, want visible values when show-env is enabled", changes[0])
	}
}

func TestComputeUpsertDiff_DoesNotRedactNonMatchingPaths(t *testing.T) {
	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  user.key: value\n"

	res := &config.Resource{
		Base: config.Base{
			Type: "instance",
			Name: "web",
			Config: map[string]string{
				"user.key": "updated",
			},
		},
		PreviewRedactPrefixes: []string{"config.environment."},
	}

	output, preview, plans := computeUpsertDiff(&Options{}, client, []*config.Resource{res})
	if preview.updated != 1 {
		t.Fatalf("updated count = %d, want 1", preview.updated)
	}
	if len(plans) != 1 || plans[0].action != upsertUpdate {
		t.Fatalf("plans = %#v, want one update plan", plans)
	}
	changes := output.Groups[0].Items[0].Changes
	if len(changes) != 1 {
		t.Fatalf("changes = %#v, want 1 change", changes)
	}
	if changes[0].Old != "value" || changes[0].New != "updated" {
		t.Fatalf("change = %#v, want visible values", changes[0])
	}
}

func TestExecutorUpsert_CreateOnlyFieldsSkipsResourceWithoutReplace(t *testing.T) {
	dir := t.TempDir()
	// Image changed (create-only) + config changed (normal) — entire resource skipped.
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.20\nconfig:\n  user.key: updated\n")

	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  user.incus-apply.created: \"true\"\n  user.incus-apply.current: |\n    image: images:alpine/3.19\n    config:\n      user.key: value\n"
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Yes: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	// No update — entire resource skipped because of create-only field change.
	if len(client.updateCalls) != 0 {
		t.Fatalf("update calls = %v, want none", client.updateCalls)
	}
	if len(renderer.outputs) != 1 {
		t.Fatalf("renderer outputs = %d, want 1", len(renderer.outputs))
	}
	// Diff changes are still shown (so user sees what changed).
	items := renderer.outputs[0].Groups[0].Items
	if len(items) != 1 {
		t.Fatalf("output items = %d, want 1", len(items))
	}
	if len(items[0].Changes) == 0 {
		t.Fatal("expected diff changes to be shown even when skipping")
	}
}

func TestExecutorUpsert_ReplaceRecreatesManagedResource(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, "instance.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.20\nconfig:\n  user.key: value\n")

	client := newFakeClient()
	client.exists["default:instance/web"] = true
	client.current["default:instance/web"] = "config:\n  user.incus-apply.created: \"true\"\n  user.incus-apply.current: |\n    image: images:alpine/3.19\n    config:\n      user.key: value\n"
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{path}, Replace: true, Yes: true, Launch: true, Quiet: true}, client, renderer)

	if err := executor.Upsert(); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if len(client.updateCalls) != 0 {
		t.Fatalf("update calls = %v, want none", client.updateCalls)
	}
	if len(client.deleteCalls) != 1 || client.deleteCalls[0] != "default:instance/web" {
		t.Fatalf("delete calls = %v, want [default:instance/web]", client.deleteCalls)
	}
	if len(client.createCalls) != 1 || client.createCalls[0] != "default:instance/web" {
		t.Fatalf("create calls = %v, want [default:instance/web]", client.createCalls)
	}
	if len(client.startCalls) != 1 || client.startCalls[0] != "default:instance/web" {
		t.Fatalf("start calls = %v, want [default:instance/web]", client.startCalls)
	}
	if len(renderer.outputs) != 1 {
		t.Fatalf("renderer outputs = %d, want 1", len(renderer.outputs))
	}
	if got := renderer.outputs[0].Summary; got != "Summary: 1 to replace." {
		t.Fatalf("summary = %q, want %q", got, "Summary: 1 to replace.")
	}
	if got := renderer.outputs[0].Groups[0].Action; got != ActionReplace {
		t.Fatalf("action = %q, want %q", got, ActionReplace)
	}
}

func TestExecutorUpsert_DuplicateResourcesSameProjectFails(t *testing.T) {
	dir := t.TempDir()
	writeConfigFile(t, dir, "one.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\n")
	writeConfigFile(t, dir, "two.incus.yaml", "type: instance\nname: web\nimage: images:alpine/3.19\n")

	client := newFakeClient()
	renderer := &captureRenderer{}
	executor := NewExecutor(Options{Files: []string{dir}, Recursive: true, Yes: true, Quiet: true}, client, renderer)

	err := executor.Upsert()
	if err == nil {
		t.Fatal("Upsert() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "default:instance/web") {
		t.Fatalf("Upsert() error = %q, want duplicate scoped id", err.Error())
	}
	if len(renderer.outputs) != 0 {
		t.Fatalf("renderer outputs = %d, want 0", len(renderer.outputs))
	}
}
