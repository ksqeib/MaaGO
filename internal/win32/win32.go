// Package win32 locates the Arknights PC client window and maps the Win32
// control options used by MaaCore's AsstAttachWindow API.
package win32

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Window is a top-level visible window.
type Window struct {
	Handle    uintptr
	Title     string
	ProcessID uint32
}

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetWindowTextLengthW     = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
)

// GameWindowTitle returns the PC client window title for a MAA client type.
func GameWindowTitle(clientType string) string {
	switch clientType {
	case "YoStarEN":
		return "Arknights"
	case "YoStarJP":
		return "アークナイツ"
	case "YoStarKR":
		return "명일방주"
	default:
		// Official, Bilibili and txwy all use the CN client window title.
		return "明日方舟"
	}
}

// FindGameWindow resolves the window title (or a user override) and returns the
// first visible match. The error message is suitable for the run log.
func FindGameWindow(clientType, override string) (Window, error) {
	title := strings.TrimSpace(override)
	if title == "" {
		title = GameWindowTitle(clientType)
	}
	matches, err := FindByTitle(title)
	if err != nil {
		return Window{}, err
	}
	if len(matches) == 0 {
		return Window{}, fmt.Errorf("未找到窗口 %q，请先启动 PC 客户端", title)
	}
	return matches[0], nil
}

// FindByTitle enumerates visible top-level windows. Exact matches are preferred;
// when none exist it falls back to a case-insensitive substring match.
func FindByTitle(title string) ([]Window, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, nil
	}
	exact, err := enumerate(title, false)
	if err != nil {
		return nil, err
	}
	if len(exact) > 0 {
		return exact, nil
	}
	return enumerate(title, true)
}

func enumerate(target string, contains bool) ([]Window, error) {
	results := make([]Window, 0, 2)
	targetLower := strings.ToLower(target)
	callback := windows.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		length, _, _ := procGetWindowTextLengthW.Call(hwnd)
		if length == 0 {
			return 1
		}
		buffer := make([]uint16, length+1)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
		title := windows.UTF16ToString(buffer)
		matched := title == target
		if contains {
			matched = strings.Contains(strings.ToLower(title), targetLower)
		}
		if !matched {
			return 1
		}
		var pid uint32
		procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		results = append(results, Window{Handle: hwnd, Title: title, ProcessID: pid})
		return 1
	})
	result, _, callErr := procEnumWindows.Call(callback, 0)
	if result == 0 {
		return nil, fmt.Errorf("EnumWindows: %v", callErr)
	}
	return results, nil
}

// Win32 screencap methods (AsstCaller.h).
const (
	ScreencapFramePool        uint64 = 2
	ScreencapDesktopDupWindow uint64 = 8
	ScreencapPrintWindow      uint64 = 16
	ScreencapScreenDC         uint64 = 32
)

// Win32 mouse methods (AsstCaller.h).
const (
	MouseSeize                    uint64 = 1
	MouseSendMessage              uint64 = 2
	MousePostMessage              uint64 = 4
	MouseSendMessageWithCursorPos uint64 = 32
	MousePostMessageWithCursorPos uint64 = 64
	MouseSendMessageWithWindowPos uint64 = 128
	MousePostMessageWithWindowPos uint64 = 256
	MouseInterception             uint64 = 512
	MouseAnchoredTouch            uint64 = 1024
)

// Win32 keyboard methods (AsstCaller.h).
const (
	KeyboardSeize       uint64 = 1
	KeyboardSendMessage uint64 = 2
	KeyboardPostMessage uint64 = 4
)

// ParseScreencap maps a config value to the core enum and a display label.
func ParseScreencap(value string) (uint64, string) {
	switch normalize(value) {
	case "desktopdupwindow", "desktopdup":
		return ScreencapDesktopDupWindow, "DesktopDupWindow"
	case "framepool":
		return ScreencapFramePool, "FramePool"
	case "screendc":
		return ScreencapScreenDC, "ScreenDC"
	default:
		return ScreencapPrintWindow, "PrintWindow"
	}
}

// ParseMouse maps a config value to the core enum and a display label.
func ParseMouse(value string) (uint64, string) {
	switch normalize(value) {
	case "seize":
		return MouseSeize, "Seize"
	case "sendmessage":
		return MouseSendMessage, "SendMessage"
	case "postmessage":
		return MousePostMessage, "PostMessage"
	case "sendmessagewithcursorpos":
		return MouseSendMessageWithCursorPos, "SendMessageWithCursorPos"
	case "postmessagewithcursorpos":
		return MousePostMessageWithCursorPos, "PostMessageWithCursorPos"
	case "sendmessagewithwindowpos":
		return MouseSendMessageWithWindowPos, "SendMessageWithWindowPos"
	case "interception":
		return MouseInterception, "Interception"
	case "anchoredtouch":
		return MouseAnchoredTouch, "AnchoredTouch"
	default:
		return MousePostMessageWithWindowPos, "PostMessageWithWindowPos"
	}
}

// ParseKeyboard maps a config value to the core enum and a display label.
func ParseKeyboard(value string) (uint64, string) {
	switch normalize(value) {
	case "seize":
		return KeyboardSeize, "Seize"
	case "postmessage":
		return KeyboardPostMessage, "PostMessage"
	default:
		return KeyboardSendMessage, "SendMessage"
	}
}

func normalize(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", ""))
}
