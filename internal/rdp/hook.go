package rdp

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MaaWin32ControlUnit captures with PrintWindow(PW_CLIENTONLY |
// PW_RENDERFULLCONTENT). PW_RENDERFULLCONTENT reads the DWM surface, which is
// black when the local desktop is locked or the window is not composed. We hook
// the control unit's PrintWindow import so captures of our proxy window go to
// the frame buffer directly; all other calls pass through.
var (
	hookOnce            sync.Once
	hookErr             error
	hookModule          windows.Handle
	originalPrintWindow uintptr
	procVirtualProtect  = windows.NewLazySystemDLL("kernel32.dll").NewProc("VirtualProtect")
	printWindowCallback = windows.NewCallback(hookPrintWindow)
)

const (
	imageDirectoryEntryImport = 1
	loadWithAlteredSearchPath = 0x00000008
	pageReadWrite             = 0x04
)

type imageImportDescriptor struct {
	OriginalFirstThunk uint32
	TimeDateStamp      uint32
	ForwarderChain     uint32
	Name               uint32
	FirstThunk         uint32
}

// InstallPrintWindowHook loads the control unit and patches its PrintWindow
// import. It is safe to call more than once.
func InstallPrintWindowHook(dllPath string) error {
	hookOnce.Do(func() {
		hookErr = installPrintWindowHook(dllPath)
	})
	return hookErr
}

func installPrintWindowHook(dllPath string) error {
	module, err := windows.LoadLibraryEx(dllPath, 0, loadWithAlteredSearchPath)
	if err != nil {
		return fmt.Errorf("加载 MaaWin32ControlUnit.dll 失败：%w", err)
	}
	hookModule = module
	user32 := windows.NewLazySystemDLL("user32.dll")
	printWindowProc := user32.NewProc("PrintWindow")
	if err := printWindowProc.Find(); err != nil {
		return err
	}
	originalPrintWindow = printWindowProc.Addr()

	base := uintptr(module)
	eLfanew := *(*uint32)(unsafe.Pointer(base + 0x3C))
	nt := base + uintptr(eLfanew)
	if *(*uint32)(unsafe.Pointer(nt)) != 0x00004550 { // PE\0\0
		return fmt.Errorf("MaaWin32ControlUnit.dll 不是有效的 PE 文件")
	}
	optional := nt + 24
	if *(*uint16)(unsafe.Pointer(optional)) != 0x20B { // PE32+
		return fmt.Errorf("MaaWin32ControlUnit.dll 不是 64 位 PE")
	}
	dataDir := optional + 112
	importRVA := *(*uint32)(unsafe.Pointer(dataDir + 8*imageDirectoryEntryImport))
	if importRVA == 0 {
		return fmt.Errorf("MaaWin32ControlUnit.dll 没有导入表")
	}
	desc := base + uintptr(importRVA)
	for {
		d := (*imageImportDescriptor)(unsafe.Pointer(desc))
		if d.Name == 0 && d.FirstThunk == 0 {
			break
		}
		dllName := windows.BytePtrToString((*byte)(unsafe.Pointer(base + uintptr(d.Name))))
		if strings.EqualFold(dllName, "user32.dll") {
			intRVA := d.OriginalFirstThunk
			if intRVA == 0 {
				intRVA = d.FirstThunk
			}
			for i := uintptr(0); ; i++ {
				nameThunkAddr := base + uintptr(intRVA) + i*8
				nameThunk := *(*uint64)(unsafe.Pointer(nameThunkAddr))
				if nameThunk == 0 {
					break
				}
				if nameThunk&(1<<63) != 0 {
					continue
				}
				hintName := base + uintptr(uint32(nameThunk)) + 2
				name := windows.BytePtrToString((*byte)(unsafe.Pointer(hintName)))
				if name != "PrintWindow" {
					continue
				}
				iatAddr := base + uintptr(d.FirstThunk) + i*8
				var oldProtect uint32
				ok, _, err := procVirtualProtect.Call(
					iatAddr,
					8,
					pageReadWrite,
					uintptr(unsafe.Pointer(&oldProtect)),
				)
				if ok == 0 {
					return fmt.Errorf("VirtualProtect 失败：%v", err)
				}
				*(*uintptr)(unsafe.Pointer(iatAddr)) = printWindowCallback
				procVirtualProtect.Call(
					iatAddr,
					8,
					uintptr(oldProtect),
					uintptr(unsafe.Pointer(&oldProtect)),
				)
				return nil
			}
		}
		desc += unsafe.Sizeof(imageImportDescriptor{})
	}
	return fmt.Errorf("未找到 PrintWindow 导入项")
}

func hookPrintWindow(hwnd, hdc, flags uintptr) uintptr {
	if w := lookupProxy(hwnd); w != nil {
		if w.session != nil {
			w.session.ensurePointerParked()
		}
		w.paint(hdc)
		return 1
	}
	if originalPrintWindow != 0 {
		r, _, _ := syscall.SyscallN(originalPrintWindow, hwnd, hdc, flags)
		return r
	}
	return 0
}
