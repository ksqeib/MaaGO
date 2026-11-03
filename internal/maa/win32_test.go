package maa

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"MaaGO/internal/win32"
)

// pcAttach is the shared setup for the opt-in PC-window tests. It loads the
// core, the PC platform overlay and attaches to the configured game window.
func pcAttach(t *testing.T, screencapValue, mouseValue string) (*Core, Handle, win32.Window) {
	t.Helper()
	title := os.Getenv("MAAGO_TEST_PC_WINDOW")
	if title == "" {
		t.Skip("set MAAGO_TEST_PC_WINDOW to the game window title")
	}
	coreDir, err := filepath.Abs(filepath.Join("..", "..", "core"))
	if err != nil {
		t.Fatal(err)
	}
	core := NewAt(coreDir)
	if !core.Available() {
		t.Skipf("MaaCore unavailable: %v", core.Err())
	}
	userDir, err := os.MkdirTemp("", "maago-pc-test-")
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

	windows, err := win32.FindByTitle(title)
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) == 0 {
		t.Fatalf("window %q not found", title)
	}
	window := windows[0]
	t.Logf("window: %s hwnd=0x%X pid=%d", window.Title, window.Handle, window.ProcessID)

	handle, err := core.Create()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { core.Destroy(handle) })

	screencap, screencapLabel := win32.ParseScreencap(screencapValue)
	mouse, mouseLabel := win32.ParseMouse(mouseValue)
	keyboard, keyboardLabel := win32.ParseKeyboard("SendMessage")
	attached, err := core.AttachWindow(handle, window.Handle, screencap, mouse, keyboard)
	if err != nil {
		t.Fatal(err)
	}
	if !attached {
		t.Fatal("AsstAttachWindow returned false")
	}
	t.Logf("attached: %s / %s / %s", screencapLabel, mouseLabel, keyboardLabel)
	return core, handle, window
}

// TestPCDiagnostics is opt-in. It attaches to an existing PC client window and
// captures one screenshot, without running any game task.
func TestPCDiagnostics(t *testing.T) {
	core, handle, _ := pcAttach(t, os.Getenv("MAAGO_TEST_PC_SCREENCAP"), "PostMessageWithWindowPos")
	if id, err := core.AsyncScreencap(handle, true); err != nil || id == 0 {
		t.Fatalf("screencap: id=%d err=%v", id, err)
	}
	buffer := make([]byte, 16*1024*1024)
	size, err := core.GetImage(handle, buffer)
	if err != nil {
		t.Fatal(err)
	}
	if size == 0 {
		t.Fatal("empty screenshot")
	}
	t.Logf("screenshot bytes: %d", size)
}

// TestPCClick is opt-in. It clicks one point in the attached window and saves a
// follow-up screenshot, which proves whether background input reaches the game.
// Set MAAGO_TEST_PC_CLICK=x,y to enable; MAAGO_TEST_PC_MOUSE overrides the mouse
// method and MAAGO_TEST_PC_SHOT sets the output path.
func TestPCClick(t *testing.T) {
	spec := os.Getenv("MAAGO_TEST_PC_CLICK")
	if spec == "" {
		t.Skip("set MAAGO_TEST_PC_CLICK=x,y to click the game window")
	}
	parts := strings.Split(spec, ",")
	if len(parts) != 2 {
		t.Fatalf("MAAGO_TEST_PC_CLICK must be x,y, got %q", spec)
	}
	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		t.Fatal(err)
	}
	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		t.Fatal(err)
	}

	mouse := os.Getenv("MAAGO_TEST_PC_MOUSE")
	if mouse == "" {
		mouse = "PostMessageWithWindowPos"
	}
	core, handle, _ := pcAttach(t, os.Getenv("MAAGO_TEST_PC_SCREENCAP"), mouse)

	if id, err := core.AsyncClick(handle, int32(x), int32(y), true); err != nil || id == 0 {
		t.Fatalf("click: id=%d err=%v", id, err)
	}
	t.Logf("clicked (%d,%d)", x, y)
	time.Sleep(3 * time.Second)

	if id, err := core.AsyncScreencap(handle, true); err != nil || id == 0 {
		t.Fatalf("screencap: id=%d err=%v", id, err)
	}
	buffer := make([]byte, 16*1024*1024)
	size, err := core.GetImage(handle, buffer)
	if err != nil {
		t.Fatal(err)
	}
	if size == 0 {
		t.Fatal("empty screenshot")
	}
	out := os.Getenv("MAAGO_TEST_PC_SHOT")
	if out == "" {
		out = filepath.Join(os.TempDir(), "maago-pc-click.png")
	}
	if err := os.WriteFile(out, buffer[:size], 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("screenshot: %s (%d bytes)", out, size)
}
