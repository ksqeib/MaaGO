package pcres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureMergesOfficialAndCustomPCTasks(t *testing.T) {
	root := t.TempDir()
	tasksDir := filepath.Join(root, "platform_diff", "PC", "resource", "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	official := `{
    "Roguelike@StageTraderInvestSystem": {
        "template": ["Roguelike@StageTraderInvestSystem.png", "Roguelike@StageTraderInvestSystemPC.png"],
        "templThreshold": 0.7
    }
}`
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.json"), []byte(official), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	if err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(tasksDir, "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tasks map[string]map[string]any
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatal(err)
	}
	if _, ok := tasks["Roguelike@StageTraderInvestSystem"]; !ok {
		t.Fatal("upstream PC override was lost")
	}
	if _, ok := tasks["QuickSwitch@Confirm"]; !ok {
		t.Fatal("custom QuickSwitch@Confirm override is missing")
	}
	for _, name := range []string{"PopupConfirmPC.png", "PopupCancelPC.png"} {
		if _, err := os.Stat(filepath.Join(root, "platform_diff", "PC", "resource", "template", name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
}

func TestEnsureExternalResourceDir(t *testing.T) {
	root := os.Getenv("MAAGO_TEST_PCRES_DIR")
	if root == "" {
		t.Skip("set MAAGO_TEST_PCRES_DIR to an installed resource directory")
	}
	if err := Ensure(root); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"PopupConfirmPC.png", "PopupCancelPC.png"} {
		if _, err := os.Stat(filepath.Join(root, "platform_diff", "PC", "resource", "template", name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "platform_diff", "PC", "resource", "tasks", "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var tasks map[string]any
	if err := json.Unmarshal(data, &tasks); err != nil {
		t.Fatal(err)
	}
	if _, ok := tasks["QuickSwitch@Confirm"]; !ok {
		t.Fatal("QuickSwitch@Confirm override is missing")
	}
}
