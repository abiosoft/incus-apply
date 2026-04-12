package incus

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/abiosoft/incus-apply/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	createdByKey    = "user.incus-apply.created"
	currentStateKey = "user.incus-apply.current"

	ManagementWarningUnmanaged = "unmanaged"
	ManagementWarningRecreate  = "recreate required"
)

// ManagementStatus reports whether a live resource is managed by incus-apply.
type ManagementStatus struct {
	Managed            bool
	Snapshot           string
	Warning            string
	UnsupportedChanges []DiffChange
}

// DiffResource computes the preview diff for a resource.
// Managed resources are diffed against the previously stored incus-apply snapshot.
// Unmanaged resources fall back to live-state diffing.
func DiffResource(currentYAML string, desired *config.Resource) ([]DiffChange, ManagementStatus, error) {
	status, err := managementStatus(currentYAML)
	if err != nil {
		return nil, ManagementStatus{}, err
	}

	if status.Managed {
		desiredSnapshot, err := managedSnapshot(desired)
		if err != nil {
			return nil, status, err
		}
		changes, err := DiffChanges(status.Snapshot, desiredSnapshot)
		previousState, parseErr := parseYAMLToMap(status.Snapshot, "stored incus-apply state")
		if parseErr != nil {
			return nil, status, parseErr
		}
		desiredState, parseErr := parseYAMLToMap(desiredSnapshot, "desired managed state")
		if parseErr != nil {
			return nil, status, parseErr
		}
		status.UnsupportedChanges = unsupportedChanges(desired.Type, changes, previousState, desiredState)
		if len(status.UnsupportedChanges) > 0 {
			status.Warning = ManagementWarningRecreate
		}
		return changes, status, err
	}

	merged, _, err := mergedConfigWithStatus(currentYAML, desired)
	if err != nil {
		return nil, status, err
	}
	changes, err := DiffChanges(currentYAML, string(merged))
	return changes, status, err
}

func managedSnapshot(res *config.Resource) (string, error) {
	state := make(map[string]any)

	if res.Image != "" {
		state["image"] = res.Image
	}
	if res.VM {
		state["vm"] = res.VM
	}
	if res.Empty {
		state["empty"] = res.Empty
	}
	if res.Storage != "" {
		state["storage"] = res.Storage
	}
	if res.Network != "" {
		state["network"] = res.Network
	}
	if res.Target != "" {
		state["target"] = res.Target
	}
	if res.ListenAddress != "" {
		state["listen_address"] = res.ListenAddress
	}
	if res.Pool != "" {
		state["pool"] = res.Pool
	}
	if res.NetworkType != "" {
		state["networkType"] = res.NetworkType
	}
	if res.Driver != "" {
		state["driver"] = res.Driver
	}
	if res.Source != "" {
		state["source"] = res.Source
	}

	if res.Config != nil {
		configState := make(map[string]any)
		for key, value := range res.Config {
			if key == createdByKey || key == currentStateKey {
				continue
			}
			configState[key] = value
		}
		if len(configState) > 0 || len(res.Config) == 0 {
			state["config"] = configState
		}
	}

	if res.Devices != nil {
		devices := make(map[string]any, len(res.Devices))
		for name, device := range res.Devices {
			deviceCopy := make(map[string]any, len(device))
			maps.Copy(deviceCopy, device)
			devices[name] = deviceCopy
		}
		state["devices"] = devices
	}

	if res.Description != "" {
		state["description"] = res.Description
	}
	if res.Profiles != nil {
		state["profiles"] = res.Profiles
	}
	if res.Ingress != nil {
		state["ingress"] = res.Ingress
	}
	if res.Egress != nil {
		state["egress"] = res.Egress
	}
	if res.Ports != nil {
		state["ports"] = res.Ports
	}

	data, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encoding managed state: %w", err)
	}
	return string(data), nil
}

func desiredForApply(res *config.Resource) (*config.Resource, string, error) {
	clone, err := cloneResource(res)
	if err != nil {
		return nil, "", err
	}

	snapshot, err := managedSnapshot(clone)
	if err != nil {
		return nil, "", err
	}

	if clone.Config == nil {
		clone.Config = map[string]string{}
	}
	clone.Config[createdByKey] = "true"
	clone.Config[currentStateKey] = snapshot

	return clone, snapshot, nil
}

func cloneResource(res *config.Resource) (*config.Resource, error) {
	data, err := yaml.Marshal(res)
	if err != nil {
		return nil, fmt.Errorf("cloning resource: %w", err)
	}

	var clone config.Resource
	if err := yaml.Unmarshal(data, &clone); err != nil {
		return nil, fmt.Errorf("cloning resource: %w", err)
	}
	clone.SourceFile = res.SourceFile
	return &clone, nil
}

func managementStatus(currentYAML string) (ManagementStatus, error) {
	current, err := parseYAMLToMap(currentYAML, "current config")
	if err != nil {
		return ManagementStatus{}, err
	}

	configMap, _ := current["config"].(map[string]any)
	if configMap == nil {
		return ManagementStatus{Warning: ManagementWarningUnmanaged}, nil
	}

	created, _ := configMap[createdByKey].(string)
	if !strings.EqualFold(strings.TrimSpace(created), "true") {
		return ManagementStatus{Warning: ManagementWarningUnmanaged}, nil
	}

	snapshot, _ := configMap[currentStateKey].(string)
	if strings.TrimSpace(snapshot) == "" {
		return ManagementStatus{Warning: ManagementWarningUnmanaged}, nil
	}

	if _, err := parseYAMLToMap(snapshot, "stored incus-apply state"); err != nil {
		return ManagementStatus{Warning: ManagementWarningUnmanaged}, nil
	}

	return ManagementStatus{Managed: true, Snapshot: snapshot}, nil
}

func mergedConfigWithStatus(currentYAML string, desired *config.Resource) ([]byte, ManagementStatus, error) {
	status, err := managementStatus(currentYAML)
	if err != nil {
		return nil, ManagementStatus{}, err
	}

	prepared, snapshot, err := desiredForApply(desired)
	if err != nil {
		return nil, status, err
	}

	if !status.Managed {
		merged, err := legacyMergeConfigs(currentYAML, prepared)
		return merged, status, err
	}

	current, err := parseYAMLToMap(currentYAML, "current config")
	if err != nil {
		return nil, status, err
	}

	previousState, err := parseYAMLToMap(status.Snapshot, "stored incus-apply state")
	if err != nil {
		return nil, status, err
	}
	newState, err := parseYAMLToMap(snapshot, "desired managed state")
	if err != nil {
		return nil, status, err
	}

	removeManagedState(current, previousState)
	applyManagedState(current, newState)
	applyTrackingState(current, snapshot)
	cleanMap(current)

	merged, err := yaml.Marshal(current)
	if err != nil {
		return nil, status, fmt.Errorf("encoding merged config: %w", err)
	}
	return merged, status, nil
}

func removeManagedState(current, previous map[string]any) {
	if previousConfig, ok := previous["config"].(map[string]any); ok {
		currentConfig, _ := current["config"].(map[string]any)
		for key := range previousConfig {
			delete(currentConfig, key)
		}
	}

	if previousDevices, ok := previous["devices"].(map[string]any); ok {
		currentDevices, _ := current["devices"].(map[string]any)
		for key := range previousDevices {
			delete(currentDevices, key)
		}
		if len(currentDevices) == 0 {
			delete(current, "devices")
		}
	}

	if _, ok := previous["description"]; ok {
		delete(current, "description")
	}
	if _, ok := previous["profiles"]; ok {
		delete(current, "profiles")
	}
	if _, ok := previous["ingress"]; ok {
		delete(current, "ingress")
	}
	if _, ok := previous["egress"]; ok {
		delete(current, "egress")
	}
	if _, ok := previous["ports"]; ok {
		delete(current, "ports")
	}
}

func applyManagedState(current, desired map[string]any) {
	if desiredConfig, ok := desired["config"].(map[string]any); ok {
		currentConfig, _ := current["config"].(map[string]any)
		if currentConfig == nil {
			currentConfig = make(map[string]any)
		}
		maps.Copy(currentConfig, desiredConfig)
		current["config"] = currentConfig
	}

	if desiredDevices, ok := desired["devices"]; ok {
		if devices, ok := desiredDevices.(map[string]any); ok {
			currentDevices, _ := current["devices"].(map[string]any)
			if currentDevices == nil {
				currentDevices = make(map[string]any)
			}
			maps.Copy(currentDevices, devices)
			current["devices"] = currentDevices
		} else {
			current["devices"] = desiredDevices
		}
	}

	for _, key := range []string{"description", "profiles", "ingress", "egress", "ports"} {
		if value, ok := desired[key]; ok {
			current[key] = value
		}
	}
}

func applyTrackingState(current map[string]any, snapshot string) {
	currentConfig, _ := current["config"].(map[string]any)
	if currentConfig == nil {
		currentConfig = make(map[string]any)
	}
	currentConfig[createdByKey] = "true"
	currentConfig[currentStateKey] = snapshot
	current["config"] = currentConfig
}

func unsupportedChanges(resourceType string, changes []DiffChange, previousState, desiredState map[string]any) []DiffChange {
	fields := createOnlyFields(resourceType)
	if len(fields) == 0 {
		return nil
	}

	var unsupported []DiffChange
	for _, change := range changes {
		root := rootPath(change.Path)
		if fields[root] {
			unsupported = append(unsupported, change)
		}
	}
	return unsupported
}

func rootPath(path string) string {
	end := len(path)
	if idx := strings.Index(path, "."); idx >= 0 && idx < end {
		end = idx
	}
	if idx := strings.Index(path, "["); idx >= 0 && idx < end {
		end = idx
	}
	return path[:end]
}

func createOnlyFields(resourceType string) map[string]bool {
	switch resourceType {
	case "instance":
		return map[string]bool{
			"image":   true,
			"vm":      true,
			"empty":   true,
			"storage": true,
			"network": true,
			"target":  true,
		}
	case "storage-pool":
		return map[string]bool{
			"driver": true,
			"source": true,
		}
	case "storage-volume", "storage-bucket":
		return map[string]bool{
			"pool": true,
		}
	case "network":
		return map[string]bool{
			"networkType": true,
		}
	case "network-forward":
		return map[string]bool{
			"network":        true,
			"listen_address": true,
		}
	default:
		return nil
	}
}
