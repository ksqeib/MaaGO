package diagnostics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"MaaGO/internal/config"
	"MaaGO/internal/maa"
)

// TestRunAgainstDevice is opt-in: it connects, takes a screenshot and taps the
// device. Set MAAGO_TEST_ADB_ADDRESS to enable it.
func TestRunAgainstDevice(t *testing.T) {
	address := os.Getenv("MAAGO_TEST_ADB_ADDRESS")
	if address == "" {
		t.Skip("set MAAGO_TEST_ADB_ADDRESS to run the device diagnostics")
	}
	coreDir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	core := maa.NewAt(coreDir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}

	cfg := config.Default()
	cfg.Connection.Emulator = "auto"
	cfg.Connection.TouchMode = "adb"
	cfg.Run.ClientType = "Official"

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	result := Run(ctx, core, cfg, address)
	for _, step := range result.Steps {
		t.Logf("%-10s %-4s %5dms %s", step.Key, step.Status, step.DurationMs, step.Detail)
	}
	if !result.OK {
		t.Fatalf("diagnostics failed: %s", result.Message)
	}
	if result.System.Resolution == "" {
		t.Fatal("resolution was not collected")
	}
	if result.System.ScreenshotPath == "" {
		t.Fatal("screenshot was not saved")
	}
	if _, err := os.Stat(result.System.ScreenshotPath); err != nil {
		t.Fatalf("screenshot file missing: %v", err)
	}
}
