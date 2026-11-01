package maa

import (
	"os"
	"testing"
)

// TestCoreConnect is opt-in because it needs a real emulator or device.
func TestCoreConnect(t *testing.T) {
	address := os.Getenv("MAAGO_TEST_ADB_ADDRESS")
	if address == "" {
		t.Skip("set MAAGO_TEST_ADB_ADDRESS to run the connection test")
	}
	adbPath := os.Getenv("MAAGO_TEST_ADB_PATH")
	if adbPath == "" {
		adbPath = ResolveADB(testCoreDir(t), "")
	}
	if adbPath == "" {
		t.Skip("adb executable not found")
	}
	if _, err := os.Stat(adbPath); err != nil {
		t.Skipf("adb not found at %s", adbPath)
	}

	core := NewAt(testCoreDir(t))
	if !core.Available() {
		t.Fatalf("MaaCore unavailable: %v", core.Err())
	}
	handle, err := core.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer core.Destroy(handle)

	if ok, err := core.SetInstanceOption(handle, InstanceOptionTouchMode, "adb"); err != nil || !ok {
		t.Fatalf("set touch mode: ok=%v err=%v", ok, err)
	}
	ok, err := core.Connect(handle, adbPath, address, "General")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if !ok {
		t.Fatal("AsstConnect returned false")
	}
}
