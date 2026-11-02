package maa

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSwitchTheme is opt-in and operates the connected device. It switches the
// game to a theme shown in the in-game theme list, then leaves it there.
func TestSwitchTheme(t *testing.T) {
	address := os.Getenv("MAAGO_TEST_ADB_ADDRESS")
	if address == "" {
		t.Skip("set MAAGO_TEST_ADB_ADDRESS")
	}
	coreDir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	core := NewAt(coreDir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	userDir, err := os.MkdirTemp("", "maago-switchtheme-user-")
	if err != nil {
		t.Fatal(err)
	}
	if err := core.SetUserDir(userDir); err != nil {
		t.Fatal(err)
	}
	handle, err := core.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer core.Destroy(handle)

	_, _ = core.SetInstanceOption(handle, InstanceOptionTouchMode, "adb")
	adbPath := ResolveADB(coreDir, "")
	if adbPath == "" {
		t.Fatal("adb not found")
	}
	if ok, err := core.Connect(handle, adbPath, address, "General"); err != nil || !ok {
		t.Fatalf("connect: ok=%v err=%v", ok, err)
	}

	id, err := core.AppendTask(handle, "SwitchTheme", `{"enable":true,"themes":["天想"]}`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("switch theme task id=%d", id)

	messages := make(chan Message, 256)
	SetMessageHandler(func(message Message) {
		select {
		case messages <- message:
		default:
		}
	})
	defer SetMessageHandler(nil)
	if ok, err := core.Start(handle); err != nil || !ok {
		t.Fatalf("start: ok=%v err=%v", ok, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	for {
		select {
		case message := <-messages:
			t.Logf("msg=%d %s", message.ID, message.Details)
			if message.ID == MsgAllTasksCompleted {
				return
			}
			if message.ID == MsgTaskChainError {
				t.Fatal("switch theme task failed")
			}
		case <-ctx.Done():
			t.Fatal("switch theme task timed out")
		}
	}
}
