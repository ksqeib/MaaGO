package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type pendingUpdate struct {
	Kind    string `json:"kind"`
	Staging string `json:"staging"`
	Version string `json:"version"`
}

func pendingPath(updatesDir string) string {
	return filepath.Join(updatesDir, "pending.json")
}

func writePending(updatesDir string, pending pendingUpdate) error {
	if err := os.MkdirAll(updatesDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(pending, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pendingPath(updatesDir), data, 0o644)
}

func readPending(updatesDir string) (pendingUpdate, bool, error) {
	data, err := os.ReadFile(pendingPath(updatesDir))
	if os.IsNotExist(err) {
		return pendingUpdate{}, false, nil
	}
	if err != nil {
		return pendingUpdate{}, false, err
	}
	var pending pendingUpdate
	if err := json.Unmarshal(data, &pending); err != nil {
		return pendingUpdate{}, false, err
	}
	return pending, true, nil
}

// MigrateLayout moves <core>/resource to the sibling resource directory used
// by the separated layout. It is a no-op when the new directory already exists.
func MigrateLayout(coreDir string) (string, error) {
	root := filepath.Dir(filepath.Clean(coreDir))
	oldResource := filepath.Join(coreDir, "resource")
	newResource := filepath.Join(root, "resource")
	if _, err := os.Stat(newResource); err == nil {
		return "", nil
	}
	if _, err := os.Stat(oldResource); err != nil {
		return "", nil
	}
	if err := os.Rename(oldResource, newResource); err != nil {
		return "", fmt.Errorf("move resource directory to %s: %w", newResource, err)
	}
	return "resource directory moved to " + newResource, nil
}

// ApplyPending installs a staged core or resource update. It must run before
// MaaCore is loaded, otherwise MaaCore.dll is locked.
func ApplyPending(coreDir string) (string, error) {
	updatesDir := filepath.Join(filepath.Dir(coreDir), ".maago-updates")
	pending, ok, err := readPending(updatesDir)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}

	switch pending.Kind {
	case "core":
		if err := validateCore(pending.Staging); err != nil {
			return "", err
		}
		backup := coreDir + ".backup"
		_ = os.RemoveAll(backup)
		hadCore := false
		if _, err := os.Stat(coreDir); err == nil {
			hadCore = true
			if err := os.Rename(coreDir, backup); err != nil {
				return "", fmt.Errorf("move old core to backup: %w", err)
			}
		}
		if err := os.Rename(pending.Staging, coreDir); err != nil {
			if hadCore {
				_ = os.Rename(backup, coreDir)
			}
			return "", fmt.Errorf("install staged core: %w", err)
		}
		if pending.Version != "" {
			_ = os.WriteFile(filepath.Join(coreDir, ".maago-core-version"), []byte(pending.Version), 0o644)
		}
		_ = os.RemoveAll(filepath.Dir(pending.Staging))
		_ = os.Remove(pendingPath(updatesDir))
		return "core " + pending.Version + " installed", nil

	case "resource":
		if _, err := os.Stat(filepath.Join(pending.Staging, "version.json")); err != nil {
			return "", fmt.Errorf("staged resource is invalid: %w", err)
		}
		target := filepath.Join(filepath.Dir(coreDir), "resource")
		if err := os.MkdirAll(target, 0o755); err != nil {
			return "", fmt.Errorf("create resource directory: %w", err)
		}
		if err := copyPath(pending.Staging, target); err != nil {
			return "", fmt.Errorf("apply staged resource: %w", err)
		}
		_ = os.RemoveAll(filepath.Dir(pending.Staging))
		_ = os.Remove(pendingPath(updatesDir))
		return "resource installed", nil

	default:
		return "", fmt.Errorf("unknown pending update kind: %s", pending.Kind)
	}
}
