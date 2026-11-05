package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestSetEnabled(t *testing.T) {
	if os.Getenv("MAAGO_TEST_AUTOSTART") == "" {
		t.Skip("set MAAGO_TEST_AUTOSTART=1 to modify the current user's Run key")
	}
	original, err := Enabled()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Set(original) }()

	if err := Set(true); err != nil {
		t.Fatal(err)
	}
	ok, err := Enabled()
	if err != nil || !ok {
		t.Fatalf("expected Run entry: ok=%v err=%v", ok, err)
	}
	command, err := CommandLine()
	if err != nil || !strings.Contains(command, "--autostart") {
		t.Fatalf("command=%q err=%v", command, err)
	}
	if err := Set(false); err != nil {
		t.Fatal(err)
	}
	ok, err = Enabled()
	if err != nil || ok {
		t.Fatalf("expected Run entry removed: ok=%v err=%v", ok, err)
	}
}
