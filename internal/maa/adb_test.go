package maa

import "testing"

func TestResolveADBAndListDevices(t *testing.T) {
	dir := testCoreDir(t)
	adb := ResolveADB(dir, "")
	if adb == "" {
		t.Skip("adb executable not found")
	}
	t.Logf("adb: %s", adb)
	for _, serial := range ADBDevices(adb) {
		t.Logf("device: %s", serial)
	}
}
