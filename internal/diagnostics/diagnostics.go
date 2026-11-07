// Package diagnostics performs a staged connection check:
// MaaCore -> ADB -> device -> connect -> system info -> screenshot -> touch.
package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/maa"
	"MaaGO/internal/rdp"
	"MaaGO/internal/win32"
)

// Step keys. The frontend localises these labels.
const (
	StepCore       = "core"
	StepADB        = "adb"
	StepDevice     = "device"
	StepConnect    = "connect"
	StepSystem     = "system"
	StepScreenshot = "screenshot"
	StepTouch      = "touch"
	StepPCUnit     = "pcunit"
	StepWindow     = "window"
	StepRDP        = "rdp"
	StepFrame      = "frame"
	StepProxy      = "proxy"
)

const (
	StatusOK   = "ok"
	StatusFail = "fail"
	StatusSkip = "skip"
)

// Step is one check result.
type Step struct {
	Key        string `json:"key"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
	DurationMs int64  `json:"durationMs"`
}

// SystemInfo is collected from adb shell and MaaCore.
type SystemInfo struct {
	Address        string `json:"address"`
	ADBPath        string `json:"adbPath"`
	Serial         string `json:"serial"`
	AndroidVersion string `json:"androidVersion"`
	Resolution     string `json:"resolution"`
	Density        string `json:"density"`
	UUID           string `json:"uuid"`
	ScreenshotPath string `json:"screenshotPath"`
	ScreenshotSize int    `json:"screenshotSize"`
}

// Result is the full diagnostic outcome.
type Result struct {
	OK      bool       `json:"ok"`
	Address string     `json:"address"`
	Steps   []Step     `json:"steps"`
	System  SystemInfo `json:"system"`
	Message string     `json:"message"`
}

var resolutionPattern = regexp.MustCompile(`(\d{3,5})\s*x\s*(\d{3,5})`)

// Run performs the staged checks. It may tap the device during the touch test.
func Run(ctx context.Context, core *maa.Core, cfg config.AppConfig, address string) Result {
	result := Result{Address: strings.TrimSpace(address)}
	steps := make([]Step, 0, 7)
	add := func(key, status, detail string, started time.Time) {
		steps = append(steps, Step{
			Key:        key,
			Status:     status,
			Detail:     detail,
			DurationMs: time.Since(started).Milliseconds(),
		})
	}
	skipRest := func(keys ...string) {
		for _, key := range keys {
			steps = append(steps, Step{Key: key, Status: StatusSkip, Detail: "blocked by an earlier step"})
		}
	}
	finish := func() Result {
		result.Steps = steps
		result.OK = true
		for _, step := range steps {
			if step.Status != StatusOK {
				result.OK = false
				if result.Message == "" {
					result.Message = step.Key + ": " + step.Detail
				}
			}
		}
		if result.OK {
			result.Message = "all checks passed"
		}
		return result
	}

	started := time.Now()
	if core == nil || !core.Available() {
		detail := "MaaCore is unavailable"
		if core != nil && core.Err() != nil {
			detail = core.Err().Error()
		}
		add(StepCore, StatusFail, detail, started)
		skipRest(StepADB, StepDevice, StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	if err := core.LoadResource(); err != nil {
		add(StepCore, StatusFail, err.Error(), started)
		skipRest(StepADB, StepDevice, StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepCore, StatusOK, core.Version(), started)
	result.System.Address = result.Address

	mode := strings.ToLower(strings.TrimSpace(cfg.Connection.Mode))
	if mode == "win32" {
		return runPC(core, cfg, &result, add, skipRest, finish)
	}
	if mode == "rdp" {
		return runRDP(core, cfg, &result, add, skipRest, finish)
	}

	started = time.Now()
	adbPath := maa.ResolveADB(core.Dir(), cfg.Connection.AdbPath)
	if adbPath == "" {
		add(StepADB, StatusFail, "adb.exe was not found", started)
		skipRest(StepDevice, StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepADB, StatusOK, adbPath, started)
	result.System.ADBPath = adbPath

	if result.Address == "" {
		devices := maa.ADBDevices(adbPath)
		if len(devices) > 0 {
			result.Address = devices[0]
		}
	}
	if result.Address == "" {
		started = time.Now()
		add(StepDevice, StatusFail, "no address was provided and no ADB device was detected", started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	result.System.Address = result.Address

	started = time.Now()
	serial := findSerial(adbPath, result.Address)
	if serial == "" {
		output, err := runADB(ctx, adbPath, "connect", result.Address)
		if err != nil && !strings.Contains(strings.ToLower(output), "connected") {
			add(StepDevice, StatusFail, truncate(strings.TrimSpace(output+" "+err.Error()), 220), started)
			skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
			return finish()
		}
		serial = findSerial(adbPath, result.Address)
	}
	if serial == "" {
		devices := maa.ADBDevices(adbPath)
		if len(devices) == 1 {
			serial = devices[0]
		}
	}
	if serial == "" {
		add(StepDevice, StatusFail, "device is not listed as 'device' by adb", started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepDevice, StatusOK, serial, started)
	result.System.Serial = serial

	started = time.Now()
	handle, err := core.Create()
	if err != nil {
		add(StepConnect, StatusFail, err.Error(), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	defer core.Destroy(handle)

	var detailMu sync.Mutex
	connectionDetails := make([]string, 0, 4)
	maa.SetMessageHandler(func(message maa.Message) {
		switch message.ID {
		case maa.MsgConnectionInfo, maa.MsgInternalError, maa.MsgInitFailed:
		default:
			return
		}
		var payload struct {
			What    string          `json:"what"`
			Details json.RawMessage `json:"details"`
		}
		_ = json.Unmarshal([]byte(message.Details), &payload)
		text := strings.TrimSpace(payload.What)
		if extra := strings.TrimSpace(string(payload.Details)); extra != "" && extra != "null" {
			text += " " + extra
		}
		if text == "" {
			text = message.Details
		}
		detailMu.Lock()
		connectionDetails = append(connectionDetails, truncate(text, 180))
		if len(connectionDetails) > 5 {
			connectionDetails = connectionDetails[len(connectionDetails)-5:]
		}
		detailMu.Unlock()
	})
	defer maa.SetMessageHandler(nil)

	touchMode := strings.TrimSpace(cfg.Connection.TouchMode)
	if touchMode == "" {
		touchMode = "adb"
	}
	_, _ = core.SetInstanceOption(handle, maa.InstanceOptionTouchMode, touchMode)
	if cfg.Run.ClientType != "" {
		_, _ = core.SetInstanceOption(handle, maa.InstanceOptionClientType, cfg.Run.ClientType)
	}
	configName := maa.ConnectionConfigName(cfg.Connection.Emulator)
	connected, err := core.Connect(handle, adbPath, serial, configName)
	if err != nil {
		add(StepConnect, StatusFail, err.Error(), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	if !connected {
		detailMu.Lock()
		reasons := strings.Join(connectionDetails, " | ")
		detailMu.Unlock()
		if reasons == "" {
			reasons = "AsstConnect returned false without a callback reason"
		}
		add(StepConnect, StatusFail, truncate(reasons, 240), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepConnect, StatusOK, fmt.Sprintf("%s · %s", configName, touchMode), started)

	started = time.Now()
	android := runShell(ctx, adbPath, serial, "getprop ro.build.version.release")
	size := runShell(ctx, adbPath, serial, "wm size")
	density := runShell(ctx, adbPath, serial, "wm density")
	uuid, _ := core.GetUUID(handle)
	resolution := parseResolution(size)
	result.System.AndroidVersion = strings.TrimSpace(android)
	result.System.Resolution = resolution
	result.System.Density = parseDensity(density)
	result.System.UUID = uuid

	systemParts := make([]string, 0, 4)
	if result.System.AndroidVersion != "" {
		systemParts = append(systemParts, "Android "+result.System.AndroidVersion)
	}
	if result.System.Resolution != "" {
		systemParts = append(systemParts, result.System.Resolution)
	}
	if result.System.Density != "" {
		systemParts = append(systemParts, result.System.Density+" dpi")
	}
	if result.System.UUID != "" {
		systemParts = append(systemParts, "UUID "+result.System.UUID)
	}
	if result.System.AndroidVersion == "" || result.System.Resolution == "" {
		detail := strings.TrimSpace(strings.Join([]string{size, density}, " | "))
		if detail == "" {
			detail = "adb shell did not return Android version / resolution"
		}
		add(StepSystem, StatusFail, truncate(detail, 220), started)
		skipRest(StepScreenshot, StepTouch)
		return finish()
	}
	add(StepSystem, StatusOK, strings.Join(systemParts, " · "), started)

	started = time.Now()
	screenshotPath, screenshotSize, err := captureScreenshot(core, handle)
	if err != nil {
		add(StepScreenshot, StatusFail, err.Error(), started)
		skipRest(StepTouch)
		return finish()
	}
	result.System.ScreenshotPath = screenshotPath
	result.System.ScreenshotSize = screenshotSize
	add(StepScreenshot, StatusOK, fmt.Sprintf("%s · %s", formatBytes(screenshotSize), screenshotPath), started)

	started = time.Now()
	if id, err := core.AsyncClick(handle, 10, 10, true); err != nil || id == 0 {
		detail := "AsstAsyncClick failed"
		if err != nil {
			detail = err.Error()
		}
		add(StepTouch, StatusFail, detail, started)
		return finish()
	}
	if _, _, err := captureScreenshot(core, handle); err != nil {
		add(StepTouch, StatusFail, "tap succeeded but the follow-up screenshot failed: "+err.Error(), started)
		return finish()
	}
	add(StepTouch, StatusOK, "tap (10,10) · follow-up screenshot ok", started)
	return finish()
}

func runRDP(
	core *maa.Core,
	cfg config.AppConfig,
	result *Result,
	add func(key, status, detail string, started time.Time),
	skipRest func(keys ...string),
	finish func() Result,
) Result {
	rc := cfg.Connection.RDP
	started := time.Now()
	session, err := rdp.Ensure(rdp.Config{
		Host:     rc.Host,
		Port:     rc.Port,
		Username: rc.Username,
		Password: rc.Password,
		Domain:   rc.Domain,
		Width:    rc.Width,
		Height:   rc.Height,
	})
	if err != nil {
		add(StepRDP, StatusFail, err.Error(), started)
		skipRest(StepFrame, StepProxy, StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	result.System.Address = fmt.Sprintf("%s@%s:%d", rc.Username, rc.Host, rc.Port)
	add(StepRDP, StatusOK, fmt.Sprintf("%s@%s:%d", rc.Username, rc.Host, rc.Port), started)

	started = time.Now()
	if !session.HasFrame() {
		add(StepFrame, StatusFail, "RDP 已连接但未收到画面；请确认远端桌面已解锁且游戏在前台", started)
		skipRest(StepProxy, StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepFrame, StatusOK, fmt.Sprintf("%dx%d", rc.Width, rc.Height), started)
	session.UnlockIfLocked(rc.Password)
	session.FocusRemote()

	started = time.Now()
	controlUnit := filepath.Join(core.Dir(), "MaaWin32ControlUnit.dll")
	if _, err := os.Stat(controlUnit); err != nil {
		add(StepProxy, StatusFail, "缺少 MaaWin32ControlUnit.dll；请重新安装 RDP 控制单元", started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	if _, err := os.Stat(filepath.Join(core.Dir(), "MaaWin32ControlUnit.official.dll")); err != nil {
		add(StepProxy, StatusFail, "当前 MaaWin32ControlUnit.dll 是官方版本；请运行 scripts\\install-rdp-controlunit.ps1 安装 RDP 控制单元", started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	bridge, err := rdp.StartBridge(session)
	if err != nil {
		add(StepProxy, StatusFail, err.Error(), started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	defer func() {
		_ = os.Unsetenv("MAAGO_RDP_BRIDGE_ADDR")
		bridge.Close()
	}()
	if err := os.Setenv("MAAGO_RDP_BRIDGE_ADDR", bridge.Addr()); err != nil {
		add(StepProxy, StatusFail, fmt.Sprintf("设置控制桥环境变量失败：%v", err), started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepProxy, StatusOK, fmt.Sprintf("无代理窗口 · 控制桥 %s", bridge.Addr()), started)

	started = time.Now()
	handle, err := core.Create()
	if err != nil {
		add(StepConnect, StatusFail, err.Error(), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	defer core.Destroy(handle)
	if cfg.Run.ClientType != "" {
		_, _ = core.SetInstanceOption(handle, maa.InstanceOptionClientType, cfg.Run.ClientType)
	}
	if err := core.LoadPlatformResource("PC"); err != nil {
		add(StepConnect, StatusFail, err.Error(), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	screencap, screencapLabel := win32.ParseScreencap("PrintWindow")
	mouse, mouseLabel := win32.ParseMouse("PostMessage")
	keyboard, keyboardLabel := win32.ParseKeyboard("SendMessage")
	attached, err := core.AttachWindow(handle, 0, screencap, mouse, keyboard)
	if err != nil || !attached {
		detail := "AsstAttachWindow 返回失败"
		if err != nil {
			detail = err.Error()
		}
		add(StepConnect, StatusFail, detail, started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepConnect, StatusOK, fmt.Sprintf("RDP 控制单元 · 截图 %s · 鼠标 %s · 键盘 %s", screencapLabel, mouseLabel, keyboardLabel), started)

	started = time.Now()
	add(StepSystem, StatusOK, fmt.Sprintf("%s@%s:%d · %dx%d", rc.Username, rc.Host, rc.Port, rc.Width, rc.Height), started)

	started = time.Now()
	screenshotPath, screenshotSize, err := captureScreenshot(core, handle)
	if err != nil {
		add(StepScreenshot, StatusFail, err.Error(), started)
		skipRest(StepTouch)
		return finish()
	}
	result.System.ScreenshotPath = screenshotPath
	result.System.ScreenshotSize = screenshotSize
	add(StepScreenshot, StatusOK, fmt.Sprintf("%s · %s", formatBytes(screenshotSize), screenshotPath), started)

	started = time.Now()
	if id, err := core.AsyncClick(handle, 10, 10, true); err != nil || id == 0 {
		detail := "AsstAsyncClick failed"
		if err != nil {
			detail = err.Error()
		}
		add(StepTouch, StatusFail, detail, started)
		return finish()
	}
	if _, _, err := captureScreenshot(core, handle); err != nil {
		add(StepTouch, StatusFail, "click succeeded but the follow-up screenshot failed: "+err.Error(), started)
		return finish()
	}
	add(StepTouch, StatusOK, "click (10,10) · follow-up screenshot ok", started)
	return finish()
}

func runPC(
	core *maa.Core,
	cfg config.AppConfig,
	result *Result,
	add func(key, status, detail string, started time.Time),
	skipRest func(keys ...string),
	finish func() Result,
) Result {
	started := time.Now()
	controlUnit := filepath.Join(core.Dir(), "MaaWin32ControlUnit.dll")
	if _, err := os.Stat(controlUnit); err != nil {
		add(StepPCUnit, StatusFail, "缺少 MaaWin32ControlUnit.dll；请从官方 MAA 发布包复制到 core\\", started)
		skipRest(StepWindow, StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepPCUnit, StatusOK, controlUnit, started)

	started = time.Now()
	window, err := win32.FindGameWindow(cfg.Run.ClientType, cfg.Connection.Win32.WindowTitle)
	if err != nil {
		add(StepWindow, StatusFail, err.Error(), started)
		skipRest(StepConnect, StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	result.System.Address = fmt.Sprintf("%s (HWND 0x%X)", window.Title, window.Handle)
	add(StepWindow, StatusOK, fmt.Sprintf("%s · HWND 0x%X · PID %d", window.Title, window.Handle, window.ProcessID), started)

	started = time.Now()
	handle, err := core.Create()
	if err != nil {
		add(StepConnect, StatusFail, err.Error(), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	defer core.Destroy(handle)
	if cfg.Run.ClientType != "" {
		_, _ = core.SetInstanceOption(handle, maa.InstanceOptionClientType, cfg.Run.ClientType)
	}
	if err := core.LoadPlatformResource("PC"); err != nil {
		add(StepConnect, StatusFail, err.Error(), started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	screencap, screencapLabel := win32.ParseScreencap(cfg.Connection.Win32.Screencap)
	mouse, mouseLabel := win32.ParseMouse(cfg.Connection.Win32.Mouse)
	keyboard, keyboardLabel := win32.ParseKeyboard(cfg.Connection.Win32.Keyboard)
	attached, err := core.AttachWindow(handle, window.Handle, screencap, mouse, keyboard)
	if err != nil || !attached {
		detail := "AsstAttachWindow 返回失败；可尝试以管理员身份运行 MaaGO"
		if err != nil {
			detail = err.Error()
		}
		add(StepConnect, StatusFail, detail, started)
		skipRest(StepSystem, StepScreenshot, StepTouch)
		return finish()
	}
	add(StepConnect, StatusOK, fmt.Sprintf("截图 %s · 鼠标 %s · 键盘 %s", screencapLabel, mouseLabel, keyboardLabel), started)

	started = time.Now()
	add(StepSystem, StatusOK, fmt.Sprintf("%s · HWND 0x%X · PID %d", window.Title, window.Handle, window.ProcessID), started)

	started = time.Now()
	screenshotPath, screenshotSize, err := captureScreenshot(core, handle)
	if err != nil {
		add(StepScreenshot, StatusFail, err.Error(), started)
		skipRest(StepTouch)
		return finish()
	}
	result.System.ScreenshotPath = screenshotPath
	result.System.ScreenshotSize = screenshotSize
	add(StepScreenshot, StatusOK, fmt.Sprintf("%s · %s", formatBytes(screenshotSize), screenshotPath), started)

	started = time.Now()
	if id, err := core.AsyncClick(handle, 10, 10, true); err != nil || id == 0 {
		detail := "AsstAsyncClick failed"
		if err != nil {
			detail = err.Error()
		}
		add(StepTouch, StatusFail, detail, started)
		return finish()
	}
	if _, _, err := captureScreenshot(core, handle); err != nil {
		add(StepTouch, StatusFail, "click succeeded but the follow-up screenshot failed: "+err.Error(), started)
		return finish()
	}
	add(StepTouch, StatusOK, "click (10,10) · follow-up screenshot ok", started)
	return finish()
}

func findSerial(adbPath, address string) string {
	devices := maa.ADBDevices(adbPath)
	for _, device := range devices {
		if device == address {
			return device
		}
	}
	return ""
}

func runADB(ctx context.Context, adbPath string, args ...string) (string, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	output, err := exec.CommandContext(cmdCtx, adbPath, args...).CombinedOutput()
	return string(output), err
}

func runShell(ctx context.Context, adbPath, serial, command string) string {
	output, _ := runADB(ctx, adbPath, "-s", serial, "shell", command)
	return strings.TrimSpace(output)
}

func parseResolution(output string) string {
	matches := resolutionPattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return ""
	}
	last := matches[len(matches)-1]
	return last[1] + "x" + last[2]
}

func parseDensity(output string) string {
	fields := strings.Fields(output)
	for i := len(fields) - 1; i >= 0; i-- {
		value := strings.TrimSpace(fields[i])
		if value != "" && strings.Trim(value, "0123456789") == "" {
			return value
		}
	}
	return ""
}

func captureScreenshot(core *maa.Core, handle maa.Handle) (string, int, error) {
	if id, err := core.AsyncScreencap(handle, true); err != nil || id == 0 {
		if err != nil {
			return "", 0, err
		}
		return "", 0, fmt.Errorf("AsstAsyncScreencap returned no id")
	}

	buffer := make([]byte, 16*1024*1024)
	size, err := core.GetImage(handle, buffer)
	if err != nil {
		return "", 0, err
	}
	if size == 0 {
		return "", 0, fmt.Errorf("screenshot is empty")
	}

	dir := apppath.DiagnosticsDir()
	path := filepath.Join(dir, "diagnostic-"+time.Now().Format("20060102-150405")+".png")
	if err := os.WriteFile(path, buffer[:size], 0o644); err != nil {
		return path, size, fmt.Errorf("screenshot captured but could not be saved: %w", err)
	}
	return path, size, nil
}

func formatBytes(size int) string {
	if size >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f KB", float64(size)/1024)
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "…"
}
