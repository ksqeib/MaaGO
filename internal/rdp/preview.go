package rdp

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	previewTitle       = "MaaGO RDP 预览（点击窗口后操作）"
	previewActiveTitle = "MaaGO RDP 预览（操作中）"
)

var (
	previewMu        sync.Mutex
	previewByHWND    = map[uintptr]*previewWindow{}
	previewClassOnce sync.Once
	previewClassErr  error
	previewWndProcFn = windows.NewCallback(previewWndProc)
)

// previewWindow is the small user-facing window. It never receives MaaCore's
// synthetic messages; those go to the hidden capture window. Physical input is
// forwarded only while this window has focus, so it stays preview-only when
// the user is working elsewhere.
type previewWindow struct {
	capture *proxyWindow
	hwnd    uintptr
	ready   chan struct{}
	done    chan struct{}
	initErr error

	interactive    bool
	swallowNext    bool
	pendingSwallow bool
	downPending    bool
	downX          uint16
	downY          uint16

	drawMu  sync.Mutex
	memDC   uintptr
	dib     uintptr
	dibBits unsafe.Pointer
	dibW    int
	dibH    int

	paintCalls atomic.Int64
	paintEmpty atomic.Int64
	stretchRet atomic.Int64
}

func newPreviewWindow(capture *proxyWindow) (*previewWindow, error) {
	p := &previewWindow{
		capture: capture,
		ready:   make(chan struct{}),
		done:    make(chan struct{}),
	}
	go p.run()
	<-p.ready
	if p.initErr != nil {
		return nil, p.initErr
	}
	return p, nil
}

func (p *previewWindow) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	previewClassOnce.Do(func() {
		className, _ := windows.UTF16PtrFromString("MaaGORDPreviewWindow")
		hInst, _, _ := procGetModuleHandleW.Call(0)
		cursor, _, _ := procLoadCursorW.Call(0, 32512)
		wc := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			Style:         0x0003,
			LpfnWndProc:   previewWndProcFn,
			HInstance:     hInst,
			HCursor:       cursor,
			LpszClassName: className,
		}
		atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			previewClassErr = fmt.Errorf("RegisterClassExW(preview): %v", err)
		}
	})
	if previewClassErr != nil {
		p.initErr = previewClassErr
		close(p.ready)
		return
	}

	className, _ := windows.UTF16PtrFromString("MaaGORDPreviewWindow")
	title, _ := windows.UTF16PtrFromString(previewTitle)
	hInst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		0x00CF0000, // WS_OVERLAPPEDWINDOW (created hidden, then shown)
		80, 80,
		496, 309,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		p.initErr = fmt.Errorf("CreateWindowExW(preview): %v", err)
		close(p.ready)
		return
	}
	p.hwnd = hwnd
	previewMu.Lock()
	previewByHWND[hwnd] = p
	previewMu.Unlock()
	procSetWindowPos.Call(hwnd, ^uintptr(0), 80, 80, 496, 309, 0x0040|0x0010) // TOPMOST | SHOWWINDOW | NOACTIVATE
	procShowWindow.Call(hwnd, 5)
	procUpdateWindow.Call(hwnd)
	close(p.ready)

	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func previewByHandle(hwnd uintptr) *previewWindow {
	previewMu.Lock()
	defer previewMu.Unlock()
	return previewByHWND[hwnd]
}

func (p *previewWindow) invalidate() {
	if p.hwnd != 0 {
		procInvalidateRect.Call(p.hwnd, 0, 1)
	}
}

func (p *previewWindow) position() {
	if p.hwnd == 0 {
		return
	}
	procSetWindowPos.Call(p.hwnd, ^uintptr(0), 80, 80, 496, 309, 0x0040|0x0010) // TOPMOST | SHOWWINDOW | NOACTIVATE
}

func (p *previewWindow) Show() {
	if p.hwnd == 0 {
		return
	}
	p.position()
	procShowWindow.Call(p.hwnd, 9) // SW_RESTORE
	procSetForeground.Call(p.hwnd)
}

func (p *previewWindow) Close() {
	previewMu.Lock()
	hwnd := p.hwnd
	previewMu.Unlock()
	if hwnd == 0 {
		return
	}
	procPostMessageW.Call(hwnd, wmProxyDestroy, 0, 0)
	<-p.done
}

func (p *previewWindow) paint(hdc uintptr) {
	if hdc == 0 || p.capture == nil {
		return
	}
	p.paintCalls.Add(1)
	p.drawMu.Lock()
	defer p.drawMu.Unlock()
	p.capture.frameMu.RLock()
	fw, fh := p.capture.frameW, p.capture.frameH
	if len(p.capture.frame) == 0 || fw <= 0 || fh <= 0 {
		p.capture.frameMu.RUnlock()
		if n := p.paintEmpty.Add(1); n <= 3 {
			log.Printf("rdp preview empty frame")
		}
		return
	}
	if p.memDC == 0 || p.dibW != fw || p.dibH != fh {
		if p.dib != 0 {
			procDeleteObject.Call(p.dib)
		}
		if p.memDC != 0 {
			procDeleteDC.Call(p.memDC)
		}
		p.memDC, p.dib, p.dibBits, p.dibW, p.dibH = 0, 0, nil, 0, 0
		mem, _, _ := procCreateCompatDC.Call(hdc)
		bi := bitmapInfo{Header: bitmapInfoHeader{
			Size:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			Width:       int32(fw),
			Height:      -int32(fh),
			Planes:      1,
			BitCount:    32,
			Compression: 0,
		}}
		var bits unsafe.Pointer
		dib, _, _ := procCreateDIBSection.Call(hdc, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
		if mem != 0 && dib != 0 && bits != nil {
			p.memDC, p.dib, p.dibBits, p.dibW, p.dibH = mem, dib, bits, fw, fh
			procSelectObject.Call(mem, dib)
		}
	}
	if p.memDC == 0 || p.dibBits == nil {
		p.capture.frameMu.RUnlock()
		return
	}
	copy(unsafe.Slice((*byte)(p.dibBits), fw*fh*4), p.capture.frame)
	var rc rect
	procGetClientRect.Call(p.hwnd, uintptr(unsafe.Pointer(&rc)))
	cw, ch := int(rc.Right-rc.Left), int(rc.Bottom-rc.Top)
	if cw <= 0 || ch <= 0 {
		cw, ch = 480, 270
	}
	p.capture.frameMu.RUnlock()
	procSetStretchBltMode.Call(hdc, 4) // HALFTONE
	r, _, _ := procStretchBlt.Call(
		hdc, 0, 0, uintptr(cw), uintptr(ch),
		p.memDC, 0, 0, uintptr(fw), uintptr(fh),
		0x00CC0020, // SRCCOPY
	)
	p.stretchRet.Store(int64(r))
	if p.paintCalls.Load() <= 3 {
		log.Printf("rdp preview stretch ret=%d %dx%d <- %dx%d", r, cw, ch, fw, fh)
	}
}

func (p *previewWindow) remotePoint(lParam uintptr) (uint16, uint16) {
	var rc rect
	procGetClientRect.Call(p.hwnd, uintptr(unsafe.Pointer(&rc)))
	cw, ch := int(rc.Right-rc.Left), int(rc.Bottom-rc.Top)
	x, y := clientXY(lParam)
	if cw <= 0 || ch <= 0 {
		cw, ch = 480, 270
	}
	return uint16(x * p.capture.session.cfg.Width / cw), uint16(y * p.capture.session.cfg.Height / ch)
}

func (p *previewWindow) setTitle(text string) {
	if p.hwnd == 0 {
		return
	}
	ptr, _ := windows.UTF16PtrFromString(text)
	procSetWindowTextW.Call(p.hwnd, uintptr(unsafe.Pointer(ptr)))
}

func previewWndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	p := previewByHandle(hwnd)
	switch uint32(message) {
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if p != nil {
			p.paint(hdc)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmPrint, wmPrintClient:
		if p != nil {
			p.paint(wParam)
		}
		return 0
	case wmEraseBkgnd:
		return 1
	case wmSize:
		if p != nil {
			p.invalidate()
		}
		return 0
	case 0x0007: // WM_SETFOCUS
		if p != nil {
			p.interactive = true
			p.swallowNext = true
			p.setTitle(previewActiveTitle)
		}
		return 0
	case 0x0008: // WM_KILLFOCUS
		if p != nil {
			p.interactive = false
			p.swallowNext = false
			p.pendingSwallow = false
			p.downPending = false
			p.setTitle(previewTitle)
		}
		return 0
	case wmLButtonDown:
		if p == nil || !p.interactive {
			return 0
		}
		if p.swallowNext {
			p.pendingSwallow = true
			return 0
		}
		p.downX, p.downY = p.remotePoint(lParam)
		p.downPending = true
		return 0
	case wmLButtonUp:
		if p == nil {
			return 0
		}
		if p.pendingSwallow {
			p.pendingSwallow = false
			p.swallowNext = false
			return 0
		}
		if p.downPending && p.interactive {
			p.capture.session.click(p.downX, p.downY)
			p.downPending = false
		}
		return 0
	case wmMouseWheel:
		if p != nil && p.interactive {
			x, y := p.remotePoint(lParam)
			delta := int16(wParam >> 16)
			amount := int16(delta / 120)
			if amount == 0 {
				if delta > 0 {
					amount = 1
				} else {
					amount = -1
				}
			}
			p.capture.session.mouseWheel(x, y, amount)
		}
		return 0
	case wmKeyDown, wmKeyUp, wmSysKeyDown, wmSysKeyUp:
		if p != nil && p.interactive && !p.pendingSwallow {
			down := uint32(message) == wmKeyDown || uint32(message) == wmSysKeyDown
			scan := byte((lParam >> 16) & 0xff)
			ext := (lParam>>24)&1 != 0
			if scan != 0 {
				p.capture.session.key(scan, down, ext)
			}
		}
		return 0
	case wmChar:
		if p != nil && p.interactive && !p.pendingSwallow {
			p.capture.session.char(rune(wParam))
		}
		return 0
	case wmClose:
		// Hide the small preview; the capture window and MaaCore stay alive.
		procShowWindow.Call(hwnd, 0)
		return 0
	case wmProxyDestroy:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		previewMu.Lock()
		delete(previewByHWND, hwnd)
		previewMu.Unlock()
		if p != nil {
			select {
			case <-p.done:
			default:
				close(p.done)
			}
			if p.dib != 0 {
				procDeleteObject.Call(p.dib)
			}
			if p.memDC != 0 {
				procDeleteDC.Call(p.memDC)
			}
		}
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}
