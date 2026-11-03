package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"MaaGO/internal/maa"
	"MaaGO/internal/rdp"
	"MaaGO/internal/win32"
)

// TestRDPProxyShot is opt-in and only validates the RDP proxy pipeline: connect,
// bind MaaCore to the proxy window, and save one MaaCore screenshot. It is much
// faster than a full task run.
func TestRDPProxyShot(t *testing.T) {
	if os.Getenv("MAAGO_TEST_RDP_SHOT") == "" {
		t.Skip("set MAAGO_TEST_RDP_SHOT=1 to capture the RDP proxy window")
	}
	host := os.Getenv("MAAGO_TEST_RDP_HOST")
	user := os.Getenv("MAAGO_TEST_RDP_USER")
	pass := os.Getenv("MAAGO_TEST_RDP_PASS")
	if host == "" || user == "" || pass == "" {
		t.Skip("set MAAGO_TEST_RDP_HOST, MAAGO_TEST_RDP_USER and MAAGO_TEST_RDP_PASS")
	}

	session, err := rdp.Ensure(rdp.Config{
		Host:     host,
		Port:     3389,
		Username: user,
		Password: pass,
		Width:    1920,
		Height:   1080,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !session.HasFrame() {
		t.Fatal("no RDP frame")
	}
	t.Logf("frame std before unlock: %.1f", session.FrameStd())
	if session.UnlockIfLocked(pass) {
		t.Logf("unlock sequence sent; std now %.1f", session.FrameStd())
	}
	time.Sleep(2 * time.Second)

	coreDir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	if err := rdp.InstallPrintWindowHook(filepath.Join(coreDir, "MaaWin32ControlUnit.dll")); err != nil {
		t.Logf("PrintWindow hook: %v", err)
	}
	core := maa.NewAt(coreDir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	userDir, err := os.MkdirTemp("", "maago-rdp-shot-")
	if err != nil {
		t.Fatal(err)
	}
	if err := core.SetUserDir(userDir); err != nil {
		t.Fatal(err)
	}
	if err := core.LoadResource(); err != nil {
		t.Fatal(err)
	}
	if err := core.LoadPlatformResource("PC"); err != nil {
		t.Fatal(err)
	}
	handle, err := core.Create()
	if err != nil {
		t.Fatal(err)
	}
	defer core.Destroy(handle)

	screencap, _ := win32.ParseScreencap("PrintWindow")
	mouse, _ := win32.ParseMouse("PostMessageWithWindowPos")
	keyboard, _ := win32.ParseKeyboard("SendMessage")
	attached, err := core.AttachWindow(handle, session.HWND(), screencap, mouse, keyboard)
	if err != nil || !attached {
		t.Fatalf("attach: attached=%v err=%v", attached, err)
	}

	if id, err := core.AsyncScreencap(handle, true); err != nil || id == 0 {
		t.Fatalf("screencap: id=%d err=%v", id, err)
	}
	buf := make([]byte, 32*1024*1024)
	size, err := core.GetImage(handle, buf)
	if err != nil {
		t.Fatal(err)
	}
	shot := filepath.Join(os.TempDir(), "maago-rdp-proxy-shot.png")
	if err := os.WriteFile(shot, buf[:size], 0o644); err != nil {
		t.Fatal(err)
	}
	frame := filepath.Join(os.TempDir(), "maago-rdp-frame.png")
	if err := session.SaveFrame(frame); err != nil {
		t.Logf("save frame: %v", err)
	}
	if data, w, h, err := session.SnapshotJPEG(640, 70); err == nil {
		t.Logf("preview jpeg: %dx%d %d bytes", w, h, len(data))
	} else {
		t.Logf("preview jpeg: %v", err)
	}
	paint, prints, stretch, empty, down, up, move := session.Stats()
	t.Logf("proxy stats: paint=%d print=%d stretch=%d empty=%d mouse=%d/%d/%d", paint, prints, stretch, empty, down, up, move)
	t.Logf("proxy shot: %s (%d bytes), frame: %s", shot, size, frame)
	time.Sleep(500 * time.Millisecond)
}
