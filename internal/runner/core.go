package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"MaaGO/internal/apppath"
	"MaaGO/internal/config"
	"MaaGO/internal/connection"
	"MaaGO/internal/corelog"
	"MaaGO/internal/maa"
	"MaaGO/internal/rdp"
	"MaaGO/internal/stages"
	"MaaGO/internal/win32"
)

type coreDetails struct {
	TaskChain     string          `json:"taskchain"`
	TaskID        int32           `json:"taskid"`
	What          string          `json:"what"`
	Details       json.RawMessage `json:"details"`
	FinishedTasks []int32         `json:"finished_tasks"`
	Subtask       string          `json:"subtask"`
	PreTask       string          `json:"pre_task"`
	First         []string        `json:"first"`
	Why           string          `json:"why"`
}

// failureTrace keeps the last recognition attempts of a task chain so a failed
// chain can explain which subtasks MaaCore could not match.
type failureTrace struct {
	first   []string
	preTask string
	seen    string
}

// runCore drives the real MaaCore queue. Completion is reported through the
// callback channel, not by polling the DLL.
func (r *Runner) runCore(ctx context.Context, tasks []config.TaskConfig, cfg config.AppConfig) {
	defer r.finishRun()

	updateTask, maaTasks := splitUpdateTask(tasks)
	total := len(tasks)
	completed := 0
	if updateTask != nil {
		options := updateTaskOptions(*updateTask)
		staged := ""
		if r.updateCheck == nil {
			r.SystemLog("warn", "资源更新：更新器未接入，跳过检查")
		} else {
			r.SystemLog("info", "资源更新：正在检查 MaaCore、资源底包和资源覆盖…")
			result, err := r.updateCheck(ctx, options)
			if err != nil {
				r.SystemLog("warn", fmt.Sprintf("资源更新：检查失败，已跳过：%v", err))
			} else {
				if options.Core {
					r.SystemLog("info", fmt.Sprintf("资源更新：MaaCore %s → %s", fallback(result.CoreCurrent, "未安装"), fallback(result.CoreLatest, "未知")))
				}
				if options.Base {
					r.SystemLog("info", fmt.Sprintf("资源更新：底包 %s → %s", fallback(result.BaseCurrent, "未安装"), fallback(result.BaseLatest, "未知")))
				}
				if options.Overlay {
					r.SystemLog("info", fmt.Sprintf("资源更新：覆盖 %s → %s", fallback(result.OverlayCurrent, "未安装"), fallback(result.OverlayLatest, "未知")))
				}
				staged = result.Staged
				if result.Message != "" {
					r.SystemLog("info", "资源更新："+result.Message)
				}
			}
		}
		r.emitTask(TaskState{TaskID: updateTask.ID, Status: TaskCompleted, Progress: 100})
		completed = 1
		r.updateCoreProgress(completed, total)
		tasks = maaTasks
		if staged != "" {
			message := "更新已暂存，请重启 MaaGO 后安装"
			if taskOptionBool(updateTask.Options, "stopOnUpdate", true) {
				r.complete(StatusStopped, message, completed, total)
				return
			}
			r.SystemLog("warn", "资源更新："+message+"；按设置继续执行剩余任务")
		}
		if len(tasks) == 0 {
			r.complete(StatusSuccess, "资源更新检查完成", completed, total)
			return
		}
	}

	if r.coreLog != nil {
		names := make([]string, 0, len(tasks))
		for _, task := range tasks {
			names = append(names, taskDisplayName(task))
		}
		version := ""
		if r.core != nil {
			version = r.core.Version()
		}
		r.coreLog.BeginSession(corelog.Session{Version: version, TaskNames: names})
		r.coreSession = true
	}

	tailCtx, tailCancel := context.WithCancel(context.Background())
	tailDone := make(chan struct{})
	go func() {
		defer close(tailDone)
		r.tailCoreFile(tailCtx)
	}()
	var tailOnce sync.Once
	stopTail := func() {
		tailOnce.Do(func() {
			tailCancel()
			<-tailDone
		})
	}
	defer stopTail()

	messages := make(chan maa.Message, 1024)
	maa.SetMessageHandler(func(message maa.Message) {
		select {
		case messages <- message:
		default:
			// Never block a MaaCore worker thread on a full UI queue.
		}
	})
	defer maa.SetMessageHandler(nil)

	fail := func(message string) {
		stopTail()
		r.SystemLog("error", message)
		r.complete(StatusFailed, message, completed, total)
	}

	if err := r.core.LoadResource(); err != nil {
		fail(fmt.Sprintf("加载 MaaCore 资源失败：%v", err))
		return
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.Connection.Mode))
	pcMode := mode == "win32"
	rdpMode := mode == "rdp"
	remoteMode := pcMode || rdpMode

	address := ""
	adbPath := ""
	configName := ""
	var pcWindow win32.Window
	var rdpSession *rdp.Session
	if remoteMode {
		if err := r.core.LoadPlatformResource("PC"); err != nil {
			fail(fmt.Sprintf("加载 PC 平台资源失败：%v", err))
			return
		}
		if pcMode {
			window, err := win32.FindGameWindow(cfg.Run.ClientType, cfg.Connection.Win32.WindowTitle)
			if err != nil {
				fail(fmt.Sprintf("查找 PC 客户端窗口失败：%v", err))
				return
			}
			pcWindow = window
			r.SystemLog("info", fmt.Sprintf("PC 客户端窗口：%s（HWND 0x%X，PID %d）", window.Title, window.Handle, window.ProcessID))
			if r.coreLog != nil {
				r.coreLog.Meta("window", window.Title)
				r.coreLog.Meta("hwnd", fmt.Sprintf("0x%X", window.Handle))
			}
		} else {
			session, err := rdp.Ensure(rdp.Config{
				Host:     cfg.Connection.RDP.Host,
				Port:     cfg.Connection.RDP.Port,
				Username: cfg.Connection.RDP.Username,
				Password: cfg.Connection.RDP.Password,
				Domain:   cfg.Connection.RDP.Domain,
				Width:    cfg.Connection.RDP.Width,
				Height:   cfg.Connection.RDP.Height,
			})
			if err != nil {
				fail(fmt.Sprintf("RDP 连接失败：%v", err))
				return
			}
			rdpSession = session
			if err := rdp.InstallPrintWindowHook(filepath.Join(r.core.Dir(), "MaaWin32ControlUnit.dll")); err != nil {
				r.SystemLog("warn", fmt.Sprintf("RDP PrintWindow 钩子安装失败：%v", err))
			}
			if rdpSession.UnlockIfLocked(cfg.Connection.RDP.Password) {
				r.SystemLog("info", "已发送 RDP 解锁按键")
			}
			rdpSession.FocusRemote()
			if wantsLaunchClient(tasks) {
				r.SystemLog("info", "开始唤醒：正在启动远端明日方舟…")
				if err := rdpSession.LaunchArknights(cfg.Connection.RDP.GamePath); err != nil {
					r.SystemLog("warn", fmt.Sprintf("开始唤醒：启动远端明日方舟失败：%v", err))
				} else {
					r.SystemLog("success", "开始唤醒：已发送启动命令，等待客户端窗口")
				}
			}
			r.SystemLog("info", fmt.Sprintf("RDP 远程桌面：%s@%s:%d（%dx%d）", cfg.Connection.RDP.Username, cfg.Connection.RDP.Host, cfg.Connection.RDP.Port, cfg.Connection.RDP.Width, cfg.Connection.RDP.Height))
			if !session.HasFrame() {
				r.SystemLog("warn", "RDP 已连接，但尚未收到远端画面")
			}
			if r.coreLog != nil {
				r.coreLog.Meta("rdp", fmt.Sprintf("%s@%s:%d", cfg.Connection.RDP.Username, cfg.Connection.RDP.Host, cfg.Connection.RDP.Port))
				r.coreLog.Meta("rdp_window", fmt.Sprintf("0x%X", session.HWND()))
			}
		}
	} else {
		address = strings.TrimSpace(cfg.Connection.Address)
		adbPath = maa.ResolveADB(r.core.Dir(), cfg.Connection.AdbPath)
		if adbPath == "" {
			fail("未找到 adb.exe，请将 platform-tools 放到 core\\adb 或在设置中填写 ADB 路径")
			return
		}

		if address == "" && cfg.Connection.AutoDetect {
			detectCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			candidates := connection.Detect(detectCtx)
			cancel()
			if len(candidates) > 0 {
				address = candidates[0].Address
				r.SystemLog("info", fmt.Sprintf("自动检测到模拟器：%s（%s）", candidates[0].Name, address))
			}
		}
		if address == "" && cfg.Connection.AutoDetect {
			if devices := maa.ADBDevices(adbPath); len(devices) > 0 {
				address = devices[0]
				r.SystemLog("info", fmt.Sprintf("自动检测到 ADB 设备：%s", address))
			}
		}
		if address == "" {
			fail("未设置连接地址，请先在设置中填写模拟器 ADB 地址")
			return
		}
	}

	handle, err := r.core.Create()
	if err != nil {
		fail(fmt.Sprintf("创建 MaaCore 实例失败：%v", err))
		return
	}
	defer r.core.Destroy(handle)

	if cfg.Run.ClientType != "" {
		_, _ = r.core.SetInstanceOption(handle, maa.InstanceOptionClientType, cfg.Run.ClientType)
	}

	if remoteMode {
		controlUnit := filepath.Join(r.core.Dir(), "MaaWin32ControlUnit.dll")
		if _, err := os.Stat(controlUnit); err != nil {
			fail("缺少 MaaWin32ControlUnit.dll（PC 窗口控制组件），请从官方 MAA 发布包复制到 core\\")
			return
		}
		var targetHWND uintptr
		var screencap, mouse, keyboard uint64
		var screencapLabel, mouseLabel, keyboardLabel string
		if pcMode {
			targetHWND = pcWindow.Handle
			screencap, screencapLabel = win32.ParseScreencap(cfg.Connection.Win32.Screencap)
			mouse, mouseLabel = win32.ParseMouse(cfg.Connection.Win32.Mouse)
			keyboard, keyboardLabel = win32.ParseKeyboard(cfg.Connection.Win32.Keyboard)
			r.SystemLog("info", fmt.Sprintf("绑定 PC 窗口：截图 %s，鼠标 %s，键盘 %s", screencapLabel, mouseLabel, keyboardLabel))
		} else {
			if rdpSession == nil || rdpSession.HWND() == 0 {
				fail("RDP 预览窗口尚未创建")
				return
			}
			targetHWND = rdpSession.HWND()
			screencap, screencapLabel = win32.ScreencapPrintWindow, "PrintWindow"
			mouse, mouseLabel = win32.MousePostMessage, "PostMessage"
			keyboard, keyboardLabel = win32.KeyboardSendMessage, "SendMessage"
			r.SystemLog("info", fmt.Sprintf("绑定 RDP 代理窗口：截图 %s，鼠标 %s，键盘 %s", screencapLabel, mouseLabel, keyboardLabel))
		}
		if r.coreLog != nil {
			r.coreLog.Meta("screencap", screencapLabel)
			r.coreLog.Meta("mouse", mouseLabel)
			r.coreLog.Meta("keyboard", keyboardLabel)
		}
		attached, err := r.core.AttachWindow(handle, targetHWND, screencap, mouse, keyboard)
		if err != nil || !attached {
			if err == nil {
				err = fmt.Errorf("AsstAttachWindow 返回失败")
			}
			if rdpMode {
				fail(fmt.Sprintf("绑定 RDP 代理窗口失败：%v", err))
			} else {
				fail(fmt.Sprintf("绑定 PC 客户端窗口失败：%v（可尝试以管理员身份运行 MaaGO）", err))
			}
			return
		}
		if rdpMode {
			rdpSession.EnsureVisible()
			r.SystemLog("success", "RDP 代理窗口绑定成功")
		} else {
			r.SystemLog("success", "PC 窗口绑定成功")
		}
	} else {
		touchMode := strings.TrimSpace(cfg.Connection.TouchMode)
		if touchMode == "" {
			touchMode = "minitouch"
		}
		if cfg.Connection.Emulator == "mumu" && cfg.Connection.ScreenshotEnhance {
			touchMode = "MumuExtras"
		}
		if ok, err := r.core.SetInstanceOption(handle, maa.InstanceOptionTouchMode, touchMode); err != nil || !ok {
			r.SystemLog("warn", fmt.Sprintf("设置触控模式失败：%v", err))
		}

		configName = maa.ConnectionConfigName(cfg.Connection.Emulator)
		r.SystemLog("info", fmt.Sprintf("连接设备：%s（配置：%s）", address, configName))
		r.SystemLog("info", fmt.Sprintf("ADB：%s", adbPath))
		if r.coreLog != nil {
			r.coreLog.Meta("address", address)
			r.coreLog.Meta("adb", adbPath)
		}
		connected, err := r.core.Connect(handle, adbPath, address, configName)
		if err != nil || !connected {
			if err == nil {
				err = fmt.Errorf("AsstConnect 返回失败")
			}
			fail(fmt.Sprintf("连接设备失败：%v", err))
			return
		}
		r.SystemLog("success", "设备连接成功")
	}

	taskByType := make(map[config.TaskType]config.TaskConfig, len(tasks))
	for _, task := range tasks {
		stageOverride := ""
		if task.Type == config.TaskFight {
			cachePath := filepath.Join(apppath.CacheDir(), stages.ActivityCacheFile)
			stageOverride, note := stages.Resolve(r.core.ResourceDir(), cfg.Run.ClientType, cachePath, task.Options, time.Now())
			if stageOverride != "" || note != "" {
				message := "作战关卡：" + fallback(stageOverride, "当前/上次")
				if plan := stages.Ordered(task.Options); len(plan) > 1 {
					message += "（顺序：" + strings.Join(plan, " > ") + "）"
				}
				if note != "" {
					message += "；" + note
				}
				r.SystemLog("info", message)
			}
		}
		params, err := maa.BuildTaskParamsWithStage(task, cfg, stageOverride)
		if err != nil {
			fail(fmt.Sprintf("生成任务参数失败（%s）：%v", taskDisplayName(task), err))
			return
		}
		taskType := maa.TaskTypeName(task.Type)
		if taskType == "" {
			fail(fmt.Sprintf("不支持的任务类型：%s", task.Type))
			return
		}
		taskID, err := r.core.AppendTask(handle, taskType, params)
		if err != nil {
			fail(fmt.Sprintf("添加任务失败（%s）：%v", taskDisplayName(task), err))
			return
		}
		taskByType[task.Type] = task
		r.SystemLog("info", fmt.Sprintf("已添加任务：%s（#%d）", taskDisplayName(task), taskID))
	}

	started, err := r.core.Start(handle)
	if err != nil || !started {
		if err == nil {
			err = fmt.Errorf("AsstStart 返回失败")
		}
		fail(fmt.Sprintf("启动任务队列失败：%v", err))
		return
	}
	r.SystemLog("info", "MaaCore 任务队列已启动")

	failedTasks := 0
	anyError := false
	stopRequested := false
	var stopTimer <-chan time.Time
	finished := map[string]bool{}
	traces := map[string]*failureTrace{}

	handleMessage := func(message maa.Message) (done bool) {
		var details coreDetails
		if message.Details != "" {
			_ = json.Unmarshal([]byte(message.Details), &details)
		}
		task, hasTask := taskByTypeByChain(taskByType, details.TaskChain)
		taskName := ""
		if hasTask {
			taskName = taskDisplayName(task)
		}

		switch message.ID {
		case maa.MsgTaskChainStart:
			if hasTask {
				r.mu.Lock()
				r.state.CurrentTaskID = task.ID
				r.state.CurrentTaskName = taskName
				r.mu.Unlock()
				r.emitTask(TaskState{TaskID: task.ID, Status: TaskRunning})
				r.emitState(r.State())
			}
			r.SystemLog("info", fmt.Sprintf("开始：%s", fallback(taskName, details.TaskChain)))

		case maa.MsgTaskChainCompleted:
			if hasTask {
				if finished[task.ID] {
					break
				}
				finished[task.ID] = true
				completed++
				r.emitTask(TaskState{TaskID: task.ID, Status: TaskCompleted, Progress: 100})
				r.updateCoreProgress(completed+failedTasks, total)
			}
			r.SystemLog("success", fmt.Sprintf("完成：%s", fallback(taskName, details.TaskChain)))

		case maa.MsgTaskChainError:
			anyError = true
			if hasTask {
				if finished[task.ID] {
					break
				}
				finished[task.ID] = true
				failedTasks++
				r.emitTask(TaskState{TaskID: task.ID, Status: TaskError})
				r.updateCoreProgress(completed+failedTasks, total)
			}
			detail := formatExtra("", details.Details)
			if detail == "" {
				detail = summarize(details.Details)
			}
			message := fmt.Sprintf("任务出错：%s", fallback(taskName, details.TaskChain))
			if detail != "" {
				message += " · " + detail
			}
			r.SystemLog("error", message)
			if trace := traces[details.TaskChain]; trace != nil && len(trace.first) > 0 {
				hint := strings.Join(trace.first, " / ")
				if trace.preTask != "" {
					hint += fmt.Sprintf("（来自 %s）", trace.preTask)
				}
				r.SystemLog("warn", fmt.Sprintf("最后识别失败：%s", hint))
			}
			if shot := r.latestDebugScreenshot(); shot != "" {
				r.SystemLog("info", fmt.Sprintf("失败截图：%s", shot))
			}

		case maa.MsgTaskChainStopped:
			if hasTask {
				if finished[task.ID] {
					break
				}
				finished[task.ID] = true
				r.emitTask(TaskState{TaskID: task.ID, Status: TaskSkipped})
			}
			r.SystemLog("warn", fmt.Sprintf("任务停止：%s", fallback(taskName, details.TaskChain)))

		case maa.MsgTaskChainExtraInfo:
			r.SystemLogCoreExtra("info", details)
		case maa.MsgSubTaskError:
			trace := traces[details.TaskChain]
			if trace == nil {
				trace = &failureTrace{}
				traces[details.TaskChain] = trace
			}
			key := strings.Join(details.First, ",") + "|" + details.PreTask
			trace.first = details.First
			trace.preTask = details.PreTask
			if key == trace.seen {
				break
			}
			trace.seen = key
			if details.TaskChain == "Recruit" && hasDetail(details.First, "RecruitNow") {
				r.SystemLog("warn", "公开招募：加急招募未成功（通常是加急许可不足或确认框未识别），已跳过；可在任务设置里关闭“加急招募”后重试")
				break
			}
			if details.TaskChain == "Recruit" && (details.What == "RecruitError" || details.Why == "recognition error") {
				r.SystemLog("warn", "公开招募识别失败：未能识别招募标签或按钮；如反复出现，请检查 resource/platform_diff/PC 的界面模板")
				break
			}
			if len(details.First) == 0 && details.PreTask == "" {
				break
			}
			text := "子任务未识别：" + strings.Join(details.First, " / ")
			if details.PreTask != "" {
				text += fmt.Sprintf("（来自 %s）", details.PreTask)
			}
			r.SystemLog("warn", text)
		case maa.MsgSubTaskExtraInfo:
			r.SystemLogCoreExtra("info", details)

		case maa.MsgConnectionInfo:
			level := "info"
			text := "连接信息"
			switch details.What {
			case "Connected":
				level = "success"
				text = "设备已连接"
			case "ConnectFailed":
				level = "error"
				text = "设备连接失败"
				anyError = true
			case "Reconnecting":
				level = "warn"
				text = "正在重新连接设备"
			case "Reconnected":
				level = "success"
				text = "设备重新连接成功"
			case "Disconnect":
				level = "warn"
				text = "设备连接已断开"
			case "UuidGot":
				text = "已获取设备 UUID"
			case "ResolutionGot":
				text = "已获取设备分辨率"
			case "ScreencapCost":
				text = "截图耗时信息"
			case "FastestWayToScreencap":
				text = "截图方式信息"
			}
			if detail := formatConnection(details.What, details.Details, details.Why); detail != "" {
				text = detail
			}
			r.SystemLog(level, text)

		case maa.MsgInitFailed:
			anyError = true
			detail := formatExtra("", details.Details)
			if detail == "" {
				detail = summarize(details.Details)
			}
			r.SystemLog("error", fmt.Sprintf("MaaCore 初始化失败：%s", detail))
		case maa.MsgInternalError:
			anyError = true
			detail := formatExtra("", details.Details)
			if detail == "" {
				detail = summarize(details.Details)
			}
			r.SystemLog("error", fmt.Sprintf("MaaCore 内部错误：%s", detail))
		case maa.MsgAllTasksCompleted:
			return true
		}
		return false
	}

loop:
	for {
		select {
		case <-ctx.Done():
			if !stopRequested {
				stopRequested = true
				r.SystemLog("warn", "收到停止请求，正在停止 MaaCore 任务…")
				if _, err := r.core.Stop(handle); err != nil {
					r.SystemLog("warn", fmt.Sprintf("停止请求失败：%v", err))
				}
				stopTimer = time.After(10 * time.Second)
			}
		case <-stopTimer:
			r.SystemLog("warn", "等待 MaaCore 停止超时，直接结束本次运行")
			break loop
		case message := <-messages:
			if handleMessage(message) {
				break loop
			}
		}
	}

	switch {
	case anyError:
		stopTail()
		r.complete(StatusFailed, "任务执行过程中出现错误", completed, total)
	case stopRequested:
		stopTail()
		r.complete(StatusStopped, "任务已停止", completed, total)
	case completed >= total:
		stopTail()
		r.complete(StatusSuccess, "全部任务完成", completed, total)
	default:
		stopTail()
		r.complete(StatusFailed, "部分任务未完成", completed, total)
	}
}

// tailCoreFile mirrors MaaCore's raw asst.log into the date-rotated core log.
// The release build logs to this file rather than stdout.
func (r *Runner) tailCoreFile(ctx context.Context) {
	if r.coreLog == nil || r.core == nil {
		return
	}
	path := r.core.DebugLogPath()
	offset := fileSize(path)
	pending := ""

	readNew := func() {
		file, err := os.Open(path)
		if err != nil {
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil {
			return
		}
		if info.Size() < offset {
			offset = 0
		}
		if info.Size() == offset {
			return
		}
		if _, err := file.Seek(offset, 0); err != nil {
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			return
		}
		offset = info.Size()
		pending += string(data)
		for {
			index := strings.IndexByte(pending, '\n')
			if index < 0 {
				break
			}
			line := strings.TrimRight(pending[:index], "\r")
			pending = pending[index+1:]
			if line != "" {
				r.coreLog.WriteLine(line)
			}
		}
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			readNew()
			if strings.TrimSpace(pending) != "" {
				r.coreLog.WriteLine(pending)
			}
			return
		case <-ticker.C:
			readNew()
		}
	}
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (r *Runner) updateCoreProgress(finished, total int) {
	if total <= 0 {
		return
	}
	r.mu.Lock()
	r.state.Completed = finished
	r.state.Progress = min(finished*100/total, 99)
	r.mu.Unlock()
	r.emitState(r.State())
}

func (r *Runner) SystemLogCoreExtra(level string, details coreDetails) {
	text := formatExtra(details.What, details.Details)
	if text == "" {
		text = summarize(details.Details)
	}
	if text == "" {
		text = strings.TrimSpace(details.What)
	}
	if text != "" {
		r.SystemLog(level, text)
	}
}

func wantsLaunchClient(tasks []config.TaskConfig) bool {
	for _, task := range tasks {
		if task.Type != config.TaskStartUp {
			continue
		}
		value, ok := task.Options["launchClient"]
		if !ok {
			return true
		}
		if enabled, ok := value.(bool); ok {
			return enabled
		}
		return true
	}
	return false
}

func taskByTypeByChain(tasks map[config.TaskType]config.TaskConfig, chain string) (config.TaskConfig, bool) {
	taskType, ok := maa.TaskTypeFromChain(chain)
	if !ok {
		return config.TaskConfig{}, false
	}
	task, ok := tasks[taskType]
	return task, ok
}

func summarize(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	text := strings.TrimSpace(string(raw))
	if len(text) > 240 {
		text = text[:240] + "…"
	}
	return text
}

// latestDebugScreenshot returns the newest MaaCore interface capture written
// during this run. MaaCore saves one whenever recognition gives up, which makes
// it the fastest way to see what the device was showing.
func (r *Runner) latestDebugScreenshot() string {
	if r.core == nil {
		return ""
	}
	dir := filepath.Join(r.core.UserDir(), "debug", "interface")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	started, _ := time.Parse(time.RFC3339, r.State().StartedAt)
	var newest string
	var newestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".png") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if !started.IsZero() && info.ModTime().Before(started.Add(-2*time.Second)) {
			continue
		}
		if info.ModTime().After(newestTime) {
			newest = filepath.Join(dir, entry.Name())
			newestTime = info.ModTime()
		}
	}
	return newest
}

func hasDetail(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func fallback(primary, secondary string) string {
	if primary != "" {
		return primary
	}
	return secondary
}
