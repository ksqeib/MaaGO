package updater

import (
	"encoding/json"
	"errors"
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

// MigrateLayout moves legacy resource directories under base/resource. Both
// cores share this tree: the custom core points MaaCore at it, and the official
// core lives in base/ next to it.
func MigrateLayout(coreDir string) (string, error) {
	root := filepath.Dir(filepath.Clean(coreDir))
	baseResource := filepath.Join(root, "base", "resource")
	if _, err := os.Stat(baseResource); err == nil {
		return "", nil
	}

	// Very old layout kept resource/ inside core/.
	oldResource := filepath.Join(coreDir, "resource")
	rootResource := filepath.Join(root, "resource")
	if _, err := os.Stat(oldResource); err == nil {
		if _, err := os.Stat(rootResource); os.IsNotExist(err) {
			if err := os.Rename(oldResource, rootResource); err != nil {
				return "", fmt.Errorf("move resource directory to %s: %w", rootResource, err)
			}
		}
	}

	if _, err := os.Stat(rootResource); err != nil {
		return "", nil
	}
	if err := os.MkdirAll(filepath.Dir(baseResource), 0o755); err != nil {
		return "", fmt.Errorf("create base directory: %w", err)
	}
	if err := os.Rename(rootResource, baseResource); err != nil {
		return "", fmt.Errorf("move resource directory to %s: %w", baseResource, err)
	}
	return "resource directory moved to " + baseResource, nil
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

	case "runtime":
		if _, err := os.Stat(filepath.Join(pending.Staging, ".maago-runtime.json")); err != nil {
			return "", fmt.Errorf("staged runtime is invalid: %w", err)
		}
		if err := copyPath(pending.Staging, coreDir); err != nil {
			return "", fmt.Errorf("install runtime dependencies: %w", err)
		}
		_ = os.RemoveAll(filepath.Dir(pending.Staging))
		_ = os.Remove(pendingPath(updatesDir))
		return "runtime dependencies " + pending.Version + " installed", nil

	case "base-package":
		if err := validateBasePackage(pending.Staging); err != nil {
			return "", err
		}
		baseDir := filepath.Join(filepath.Dir(coreDir), "base")
		backup := baseDir + ".backup"
		_ = os.RemoveAll(backup)
		hadTarget := false
		if _, err := os.Stat(baseDir); err == nil {
			hadTarget = true
			if err := os.Rename(baseDir, backup); err != nil {
				return "", fmt.Errorf("move old base to backup: %w", err)
			}
		}
		if err := os.Rename(pending.Staging, baseDir); err != nil {
			if hadTarget {
				_ = os.Rename(backup, baseDir)
			}
			return "", fmt.Errorf("install base package: %w", err)
		}
		if pending.Version != "" {
			_ = os.WriteFile(filepath.Join(baseDir, ".maago-base-version"), []byte(pending.Version), 0o644)
		}
		_ = os.RemoveAll(filepath.Dir(pending.Staging))
		_ = os.RemoveAll(backup)
		_ = os.Remove(pendingPath(updatesDir))
		return "base package " + pending.Version + " installed", nil

	case "resource-base":
		// Legacy staged resource-only base. Install it as base/resource.
		if _, err := os.Stat(filepath.Join(pending.Staging, "version.json")); err != nil {
			return "", fmt.Errorf("staged base resource is invalid: %w", err)
		}
		baseDir := filepath.Join(filepath.Dir(coreDir), "base")
		target := filepath.Join(baseDir, "resource")
		backup := target + ".backup"
		_ = os.RemoveAll(backup)
		hadTarget := false
		if _, err := os.Stat(target); err == nil {
			hadTarget = true
			if err := os.Rename(target, backup); err != nil {
				return "", fmt.Errorf("move old resource to backup: %w", err)
			}
		}
		if err := os.Rename(pending.Staging, target); err != nil {
			if hadTarget {
				_ = os.Rename(backup, target)
			}
			return "", fmt.Errorf("install base resource: %w", err)
		}
		_ = os.RemoveAll(backup)
		if pending.Version != "" {
			_ = os.WriteFile(filepath.Join(baseDir, ".maago-base-version"), []byte(pending.Version), 0o644)
		}
		_ = os.Remove(filepath.Join(target, ".maago-overlay-version"))
		_ = os.RemoveAll(filepath.Dir(pending.Staging))
		_ = os.Remove(pendingPath(updatesDir))
		return "base resource " + pending.Version + " installed", nil

	case "resource-overlay":
		if _, err := os.Stat(filepath.Join(pending.Staging, "version.json")); err != nil {
			return "", fmt.Errorf("staged resource overlay is invalid: %w", err)
		}
		target := filepath.Join(filepath.Dir(coreDir), "base", "resource")
		if _, err := os.Stat(filepath.Join(target, "version.json")); err != nil {
			return "", errors.New("cannot apply overlay: install the base package first")
		}
		if err := copyPath(pending.Staging, target); err != nil {
			return "", fmt.Errorf("apply resource overlay: %w", err)
		}
		if pending.Version != "" {
			_ = os.WriteFile(filepath.Join(target, ".maago-overlay-version"), []byte(pending.Version), 0o644)
		}
		_ = os.RemoveAll(filepath.Dir(pending.Staging))
		_ = os.Remove(pendingPath(updatesDir))
		return "resource overlay " + pending.Version + " installed", nil

	case "resource":
		if _, err := os.Stat(filepath.Join(pending.Staging, "version.json")); err != nil {
			return "", fmt.Errorf("staged resource is invalid: %w", err)
		}
		target := filepath.Join(filepath.Dir(coreDir), "base", "resource")
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
