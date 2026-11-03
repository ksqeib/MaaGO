// Package maa wraps the MaaCore C API.
//
// The DLL is loaded from a dedicated core directory next to the executable (or
// the working directory during go run). Loading through LoadLibraryEx with
// LOAD_WITH_ALTERED_SEARCH_PATH lets its sibling DLLs resolve from core/ without
// scattering them next to the launcher.
package maa

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"MaaGO/internal/apppath"

	"golang.org/x/sys/windows"
)

// Message is one MaaCore callback notification.
type Message struct {
	ID      int32
	Details string
}

// Message IDs from MaaCore.
const (
	MsgInternalError     int32 = 0
	MsgInitFailed        int32 = 1
	MsgConnectionInfo    int32 = 2
	MsgAllTasksCompleted int32 = 3
	MsgAsyncCallInfo     int32 = 4
	MsgDestroyed         int32 = 5

	MsgTaskChainError     int32 = 10000
	MsgTaskChainStart     int32 = 10001
	MsgTaskChainCompleted int32 = 10002
	MsgTaskChainExtraInfo int32 = 10003
	MsgTaskChainStopped   int32 = 10004

	MsgSubTaskError     int32 = 20000
	MsgSubTaskStart     int32 = 20001
	MsgSubTaskCompleted int32 = 20002
	MsgSubTaskExtraInfo int32 = 20003
	MsgSubTaskStopped   int32 = 20004
)

// Instance option keys.
const (
	InstanceOptionTouchMode  int32 = 2
	InstanceOptionClientType int32 = 6
)

var (
	callbackPtr = syscall.NewCallback(coreCallback)
	handlerMu   sync.RWMutex
	handler     func(Message)
)

// SetMessageHandler installs the callback sink. Passing nil removes it.
func SetMessageHandler(fn func(Message)) {
	handlerMu.Lock()
	handler = fn
	handlerMu.Unlock()
}

func coreCallback(msg uintptr, details uintptr, _ uintptr) uintptr {
	text := ""
	if details != 0 {
		text = windows.BytePtrToString((*byte)(unsafe.Pointer(details)))
	}
	handlerMu.RLock()
	fn := handler
	handlerMu.RUnlock()
	if fn != nil {
		fn(Message{ID: int32(msg), Details: text})
	}
	return 0
}

const versionMarkerFile = ".maago-core-version"

// Core owns the MaaCore DLL handle and resolved functions.
type Core struct {
	dir          string
	resourceRoot string
	userDir      string
	dll          windows.Handle
	procs        map[string]uintptr
	version      string
	loadErr      error

	resourceOnce sync.Once
	resourceErr  error
}

// Handle is an AsstHandle.
type Handle uintptr

// New loads MaaCore from a core directory. It never panics; check Available.
func New() *Core {
	dir, err := findCoreDir()
	core := &Core{dir: dir, resourceRoot: filepath.Dir(filepath.Clean(dir)), procs: map[string]uintptr{}}
	if err != nil {
		core.loadErr = err
		return core
	}
	core.load()
	return core
}

// NewAt loads MaaCore from an explicit directory.
func NewAt(dir string) *Core {
	core := &Core{dir: dir, resourceRoot: filepath.Dir(filepath.Clean(dir)), procs: map[string]uintptr{}}
	core.load()
	return core
}

// Dir returns the directory containing MaaCore.dll.
func (c *Core) Dir() string {
	if c == nil {
		return ""
	}
	return c.dir
}

// ResourceDir returns the resource directory next to the core directory.
func (c *Core) ResourceDir() string {
	if c == nil {
		return ""
	}
	return filepath.Join(c.resourceRoot, "resource")
}

// SetUserDir sets MaaCore's user directory, which holds its debug log and
// caches. It must be called before LoadResource.
func (c *Core) SetUserDir(path string) error {
	if !c.Available() {
		return c.Err()
	}
	if err := os.MkdirAll(filepath.Join(path, "debug"), 0o755); err != nil {
		return err
	}
	ok, err := c.callBool("AsstSetUserDir", path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("AsstSetUserDir rejected %s", path)
	}
	c.userDir = path
	return nil
}

// UserDir returns the configured user directory.
func (c *Core) UserDir() string {
	if c == nil {
		return ""
	}
	if c.userDir != "" {
		return c.userDir
	}
	return c.dir
}

// DebugLogPath is MaaCore's own raw log file.
func (c *Core) DebugLogPath() string {
	return filepath.Join(c.UserDir(), "debug", "asst.log")
}

// Available reports whether the DLL and all required exports loaded.
func (c *Core) Available() bool {
	return c != nil && c.loadErr == nil && c.dll != 0
}

// Err returns the load error, if any.
func (c *Core) Err() error {
	if c == nil {
		return errors.New("MaaCore is not initialised")
	}
	return c.loadErr
}

// Version returns the version reported by AsstGetVersion.
func (c *Core) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

func findCoreDir() (string, error) {
	candidates := make([]string, 0, 3)
	if env := strings.TrimSpace(os.Getenv("MAAGO_CORE_DIR")); env != "" {
		candidates = append(candidates, env)
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(exeDir, "core"),
			filepath.Join(exeDir, "..", "core"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "core"))
	}

	for _, dir := range candidates {
		if info, err := os.Stat(filepath.Join(dir, "MaaCore.dll")); err == nil && !info.IsDir() {
			return filepath.Clean(dir), nil
		}
	}
	if len(candidates) == 0 {
		return "core", errors.New("could not determine the core directory")
	}
	return candidates[0], fmt.Errorf("MaaCore.dll not found in %s", candidates[0])
}

// DefaultCoreDir returns the preferred core directory even when it does not
// exist yet. Used by the updater before MaaCore is loaded.
func DefaultCoreDir() string {
	if dir, err := findCoreDir(); err == nil {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Clean(filepath.Join(filepath.Dir(exe), "..", "core"))
	}
	if cwd, err := os.Getwd(); err == nil {
		return filepath.Join(cwd, "core")
	}
	return "core"
}

func (c *Core) load() {
	dllPath, err := filepath.Abs(filepath.Join(c.dir, "MaaCore.dll"))
	if err != nil {
		c.loadErr = err
		return
	}
	dll, err := windows.LoadLibraryEx(dllPath, 0, windows.LOAD_WITH_ALTERED_SEARCH_PATH)
	if err != nil {
		c.loadErr = fmt.Errorf("load %s: %w", dllPath, err)
		return
	}

	exports := []string{
		"AsstSetUserDir",
		"AsstLoadResource",
		"AsstSetStaticOption",
		"AsstCreateEx",
		"AsstDestroy",
		"AsstSetInstanceOption",
		"AsstConnect",
		"AsstAppendTask",
		"AsstStart",
		"AsstStop",
		"AsstRunning",
		"AsstConnected",
		"AsstGetVersion",
		"AsstAsyncScreencap",
		"AsstAsyncClick",
		"AsstGetImage",
		"AsstGetUUID",
		"AsstGetNullSize",
	}
	for _, name := range exports {
		addr, err := windows.GetProcAddress(dll, name)
		if err != nil {
			_ = windows.FreeLibrary(dll)
			c.loadErr = fmt.Errorf("resolve %s: %w", name, err)
			return
		}
		c.procs[name] = addr
	}

	c.dll = dll
	if addr := c.procs["AsstGetVersion"]; addr != 0 {
		result, _, _ := syscall.SyscallN(addr)
		if result != 0 {
			c.version = windows.BytePtrToString((*byte)(unsafe.Pointer(result)))
		}
	}
	if c.version == "" || c.version == "DEBUG_VERSION" {
		if data, err := os.ReadFile(filepath.Join(c.dir, versionMarkerFile)); err == nil {
			if marker := strings.TrimSpace(string(data)); marker != "" {
				c.version = marker
			}
		}
	}
}

// LoadResource loads the sibling resource directory once. AsstLoadResource
// appends "/resource", so pass the parent of the resource directory.
func (c *Core) LoadResource() error {
	if !c.Available() {
		return c.Err()
	}
	c.resourceOnce.Do(func() {
		if c.userDir == "" {
			if err := c.SetUserDir(filepath.Join(apppath.Root(), "core")); err != nil {
				c.resourceErr = fmt.Errorf("set default user dir: %w", err)
				return
			}
		}
		ok, err := c.callBool("AsstLoadResource", c.resourceRoot)
		if err != nil {
			c.resourceErr = err
			return
		}
		if !ok {
			c.resourceErr = fmt.Errorf("MaaCore rejected resource directory %s", c.ResourceDir())
		}
	})
	return c.resourceErr
}

// ResourceError returns the resource load error, if any.
func (c *Core) ResourceError() error {
	if c == nil {
		return errors.New("MaaCore is not initialised")
	}
	return c.resourceErr
}

// Create creates an instance and connects it to the callback sink.
func (c *Core) Create() (Handle, error) {
	if err := c.LoadResource(); err != nil {
		return 0, err
	}
	addr, err := c.proc("AsstCreateEx")
	if err != nil {
		return 0, err
	}
	result, _, _ := syscall.SyscallN(addr, callbackPtr, 0)
	if result == 0 {
		return 0, errors.New("AsstCreateEx returned a null handle")
	}
	return Handle(result), nil
}

// Destroy releases an instance.
func (c *Core) Destroy(handle Handle) {
	if !c.Available() || handle == 0 {
		return
	}
	addr, err := c.proc("AsstDestroy")
	if err != nil {
		return
	}
	_, _, _ = syscall.SyscallN(addr, uintptr(handle))
}

// SetInstanceOption sets a per-instance option.
func (c *Core) SetInstanceOption(handle Handle, key int32, value string) (bool, error) {
	valuePtr, err := windows.BytePtrFromString(value)
	if err != nil {
		return false, err
	}
	addr, err := c.proc("AsstSetInstanceOption")
	if err != nil {
		return false, err
	}
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(handle),
		uintptr(key),
		uintptr(unsafe.Pointer(valuePtr)),
	)
	return byte(result) != 0, nil
}

// Static option keys from AsstTypes.h.
const (
	StaticOptionCpuOCR int32 = 1
	StaticOptionGpuOCR int32 = 2
)

// SetStaticOption sets a process-wide MaaCore option. It must be called before
// LoadResource; OCR provider switching is not supported after resources load.
func (c *Core) SetStaticOption(key int32, value string) (bool, error) {
	valuePtr, err := windows.BytePtrFromString(value)
	if err != nil {
		return false, err
	}
	addr, err := c.proc("AsstSetStaticOption")
	if err != nil {
		return false, err
	}
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(key),
		uintptr(unsafe.Pointer(valuePtr)),
	)
	return byte(result) != 0, nil
}

// Connect performs the synchronous connection.
func (c *Core) Connect(handle Handle, adbPath, address, config string) (bool, error) {
	adbPtr, err := windows.BytePtrFromString(adbPath)
	if err != nil {
		return false, err
	}
	addressPtr, err := windows.BytePtrFromString(address)
	if err != nil {
		return false, err
	}
	configPtr, err := windows.BytePtrFromString(config)
	if err != nil {
		return false, err
	}
	addr, err := c.proc("AsstConnect")
	if err != nil {
		return false, err
	}
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(handle),
		uintptr(unsafe.Pointer(adbPtr)),
		uintptr(unsafe.Pointer(addressPtr)),
		uintptr(unsafe.Pointer(configPtr)),
	)
	return byte(result) != 0, nil
}

// AppendTask queues one task and returns its AsstTaskId.
func (c *Core) AppendTask(handle Handle, taskType, params string) (int32, error) {
	typePtr, err := windows.BytePtrFromString(taskType)
	if err != nil {
		return 0, err
	}
	paramsPtr, err := windows.BytePtrFromString(params)
	if err != nil {
		return 0, err
	}
	addr, err := c.proc("AsstAppendTask")
	if err != nil {
		return 0, err
	}
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(handle),
		uintptr(unsafe.Pointer(typePtr)),
		uintptr(unsafe.Pointer(paramsPtr)),
	)
	id := int32(result)
	if id == 0 {
		return 0, errors.New("MaaCore rejected task parameters")
	}
	return id, nil
}

// Start starts the queued tasks.
func (c *Core) Start(handle Handle) (bool, error) {
	return c.callBoolHandle("AsstStart", handle)
}

// Stop asks the running queue to stop.
func (c *Core) Stop(handle Handle) (bool, error) {
	return c.callBoolHandle("AsstStop", handle)
}

// Running reports whether the queue is running.
func (c *Core) Running(handle Handle) bool {
	ok, _ := c.callBoolHandle("AsstRunning", handle)
	return ok
}

// Connected reports whether the instance is connected.
func (c *Core) Connected(handle Handle) bool {
	ok, _ := c.callBoolHandle("AsstConnected", handle)
	return ok
}

// AsyncScreencap requests a screenshot. With block=true it returns after the
// frame is ready.
func (c *Core) AsyncScreencap(handle Handle, block bool) (int32, error) {
	addr, err := c.proc("AsstAsyncScreencap")
	if err != nil {
		return 0, err
	}
	result, _, _ := syscall.SyscallN(addr, uintptr(handle), boolPtr(block))
	return int32(result), nil
}

// AsyncClick sends one tap. With block=true it returns after the tap.
func (c *Core) AsyncClick(handle Handle, x, y int32, block bool) (int32, error) {
	addr, err := c.proc("AsstAsyncClick")
	if err != nil {
		return 0, err
	}
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(handle),
		uintptr(x),
		uintptr(y),
		boolPtr(block),
	)
	return int32(result), nil
}

// GetImage copies the cached PNG frame into buffer.
func (c *Core) GetImage(handle Handle, buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, errors.New("image buffer is empty")
	}
	addr, err := c.proc("AsstGetImage")
	if err != nil {
		return 0, err
	}
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(handle),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if result == c.nullSize() {
		return 0, errors.New("AsstGetImage returned no data")
	}
	if int(result) > len(buffer) {
		return 0, errors.New("AsstGetImage returned an oversized frame")
	}
	return int(result), nil
}

// GetUUID returns the device UUID reported by MaaCore.
func (c *Core) GetUUID(handle Handle) (string, error) {
	addr, err := c.proc("AsstGetUUID")
	if err != nil {
		return "", err
	}
	buffer := make([]byte, 256)
	result, _, _ := syscall.SyscallN(
		addr,
		uintptr(handle),
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if result == c.nullSize() || result == 0 {
		return "", errors.New("AsstGetUUID returned no data")
	}
	if int(result) > len(buffer) {
		return "", errors.New("AsstGetUUID returned an oversized value")
	}
	return strings.TrimRight(string(buffer[:result]), "\x00"), nil
}

func (c *Core) nullSize() uintptr {
	addr, err := c.proc("AsstGetNullSize")
	if err != nil {
		return ^uintptr(0)
	}
	result, _, _ := syscall.SyscallN(addr)
	return result
}

func boolPtr(value bool) uintptr {
	if value {
		return 1
	}
	return 0
}

func (c *Core) callBool(name, value string) (bool, error) {
	valuePtr, err := windows.BytePtrFromString(value)
	if err != nil {
		return false, err
	}
	addr, err := c.proc(name)
	if err != nil {
		return false, err
	}
	result, _, _ := syscall.SyscallN(addr, uintptr(unsafe.Pointer(valuePtr)))
	return byte(result) != 0, nil
}

func (c *Core) callBoolHandle(name string, handle Handle) (bool, error) {
	addr, err := c.proc(name)
	if err != nil {
		return false, err
	}
	result, _, _ := syscall.SyscallN(addr, uintptr(handle))
	return byte(result) != 0, nil
}

func (c *Core) proc(name string) (uintptr, error) {
	if !c.Available() {
		return 0, c.Err()
	}
	addr, ok := c.procs[name]
	if !ok || addr == 0 {
		return 0, fmt.Errorf("MaaCore export %s is unavailable", name)
	}
	return addr, nil
}
