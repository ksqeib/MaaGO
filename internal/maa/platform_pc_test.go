package maa

import (
	"os"
	"testing"
)

func TestLoadPlatformPCResource(t *testing.T) {
	if os.Getenv("MAAGO_TEST_PC_RESOURCE") == "" {
		t.Skip("set MAAGO_TEST_PC_RESOURCE=1 and MAAGO_TEST_CORE_DIR to test the PC overlay")
	}
	dir := os.Getenv("MAAGO_TEST_CORE_DIR")
	if dir == "" {
		t.Skip("set MAAGO_TEST_CORE_DIR to an installed core directory")
	}
	core := NewAt(dir)
	if err := core.Err(); err != nil {
		t.Fatal(err)
	}
	userDir, err := os.MkdirTemp("", "maago-pc-resource-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(userDir)
	if err := core.SetUserDir(userDir); err != nil {
		t.Fatal(err)
	}
	if err := core.LoadResource(); err != nil {
		t.Fatal(err)
	}
	if err := core.LoadPlatformResource("PC"); err != nil {
		t.Fatal(err)
	}
}
