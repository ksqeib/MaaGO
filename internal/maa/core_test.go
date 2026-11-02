package maa

import (
	"os"
	"path/filepath"
	"testing"

	"MaaGO/internal/config"
)

func testCoreDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "MaaCore.dll")); err != nil {
		t.Skip("core directory is not installed")
	}
	return dir
}

func TestCoreLoadsAndCreatesInstance(t *testing.T) {
	core := NewAt(testCoreDir(t))
	if !core.Available() {
		t.Fatalf("MaaCore unavailable: %v", core.Err())
	}
	if core.Version() == "" {
		t.Fatal("AsstGetVersion returned an empty version")
	}
	t.Logf("MaaCore version: %s", core.Version())
	userDir, err := os.MkdirTemp("", "maago-core-test-")
	if err != nil {
		t.Fatal(err)
	}
	if err := core.SetUserDir(userDir); err != nil {
		t.Fatalf("set user dir: %v", err)
	}
	if err := core.LoadResource(); err != nil {
		t.Fatalf("load resource: %v", err)
	}

	handle, err := core.Create()
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	defer core.Destroy(handle)

	cfg := config.Default()
	for _, task := range cfg.Tasks {
		params, err := BuildTaskParams(task, cfg)
		if err != nil {
			t.Fatalf("%s: %v", task.Type, err)
		}
		taskType := TaskTypeName(task.Type)
		id, err := core.AppendTask(handle, taskType, params)
		if err != nil {
			t.Fatalf("%s append: %v (params=%s)", task.Type, err, params)
		}
		if id == 0 {
			t.Fatalf("%s append returned zero id", task.Type)
		}
	}
}

func TestBuildTaskParams(t *testing.T) {
	cfg := config.Default()
	for _, task := range cfg.Tasks {
		params, err := BuildTaskParams(task, cfg)
		if err != nil {
			t.Fatalf("%s: %v", task.Type, err)
		}
		if params == "" || params[0] != '{' {
			t.Fatalf("%s: unexpected params %q", task.Type, params)
		}
	}
}
