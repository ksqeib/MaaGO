// Package pcres carries small PC-platform resource overlays that may be absent
// after a MaaResource update. The MaaResource main branch has no platform_diff
// directory, so a resource refresh can remove the PC overlay. Ensure restores
// the official PC task overrides plus the PC "聘用候选人" template.
package pcres

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

//go:embed RecruitFinish.png
var recruitFinish []byte

//go:embed PC-tasks.json
var pcTasks []byte

//go:embed PC-overrides.json
var pcOverrides []byte

//go:embed PopupConfirmPC.png
var popupConfirmPC []byte

//go:embed PopupCancelPC.png
var popupCancelPC []byte

//go:embed Roguelike@StageTraderInvestSystemPC.png
var roguelikePC []byte

var assets = []struct {
	rel  string
	data []byte
	// legacySHA256 is the bundled file this asset replaces. An installed file
	// with another hash is preserved, so a newer upstream template wins.
	legacySHA256 string
}{
	{"platform_diff/PC/resource/tasks/tasks.json", pcTasks, ""},
	{"platform_diff/PC/resource/template/Roguelike/base/Roguelike@StageTraderInvestSystemPC.png", roguelikePC, ""},
	// The 2026-11 recruit UI widened the "聘用候选人" button; replace only the
	// template MaaGO used to ship.
	{"platform_diff/PC/resource/template/Recruit/RecruitFinish.png", recruitFinish, "ee36e3475142bb9c1c6f0720a721e1575bb5a816a0ad3459f7bed2b97a14967f"},
	{"platform_diff/PC/resource/template/PopupConfirmPC.png", popupConfirmPC, ""},
	{"platform_diff/PC/resource/template/PopupCancelPC.png", popupCancelPC, ""},
}

func fileSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func shouldReplace(existing []byte, legacySHA256 string) bool {
	return legacySHA256 != "" && fileSHA256(existing) == legacySHA256
}

// Ensure restores missing PC overlay files. Existing files are preserved,
// except for bundled templates that match a known outdated hash.
func Ensure(resourceDir string) error {
	if resourceDir == "" {
		return nil
	}
	for _, asset := range assets {
		path := filepath.Join(resourceDir, asset.rel)
		if _, err := os.Stat(path); err == nil {
			if asset.legacySHA256 == "" {
				continue
			}
			existing, readErr := os.ReadFile(path)
			if readErr != nil || !shouldReplace(existing, asset.legacySHA256) {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, asset.data, 0o644); err != nil {
			return err
		}
	}
	return mergeTasks(filepath.Join(resourceDir, "platform_diff/PC/resource/tasks/tasks.json"), pcOverrides)
}

// mergeTasks adds MaaGO's PC task overrides without discarding upstream PC
// overrides. It must run before LoadPlatformResource so MaaCore compiles the
// merged task graph.
func mergeTasks(path string, overlay []byte) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var tasks map[string]json.RawMessage
	if err := json.Unmarshal(existing, &tasks); err != nil {
		return err
	}
	var extra map[string]json.RawMessage
	if err := json.Unmarshal(overlay, &extra); err != nil {
		return err
	}
	changed := false
	for name, value := range extra {
		if current, ok := tasks[name]; ok && bytes.Equal(current, value) {
			continue
		}
		tasks[name] = value
		changed = true
	}
	if !changed {
		return nil
	}
	data, err := json.MarshalIndent(tasks, "", "    ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
