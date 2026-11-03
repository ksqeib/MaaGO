package rdp

import (
	"fmt"
	"image"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/casvisor/gordp"
	"golang.org/x/sys/windows"
)

const (
	wmDestroy      = 0x0002
	wmSize         = 0x0005
	wmPaint        = 0x000F
	wmClose        = 0x0010
	wmEraseBkgnd   = 0x0014
	wmMouseMove    = 0x0200
	wmLButtonDown  = 0x0201
	wmLButtonUp    = 0x0202
	wmRButtonDown  = 0x0204
	wmRButtonUp    = 0x0205
	wmMouseWheel   = 0x020A
	wmKeyDown      = 0x0100
	wmKeyUp        = 0x0101
	wmChar         = 0x0102
	wmSysKeyDown   = 0x0104
	wmSysKeyUp     = 0x0105
	wmPrint        = 0x0317
	wmPrintClient  = 0x0318
	wmProxyDestroy = 0x8001 // WM_APP + 1, internal destroy request
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")

	procRegisterClassExW  = user32.NewProc("RegisterClassExW")
	procCreateWindowExW   = user32.NewProc("CreateWindowExW")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procPostMessageW      = user32.NewProc("PostMessageW")
	procPostQuitMessage   = user32.NewProc("PostQuitMessage")
	procShowWindow        = user32.NewProc("ShowWindow")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procSetWindowTextW    = user32.NewProc("SetWindowTextW")
	procSetForeground     = user32.NewProc("SetForegroundWindow")
	procUpdateWindow      = user32.NewProc("UpdateWindow")
	procGetMessageW       = user32.NewProc("GetMessageW")
	procTranslateMessage  = user32.NewProc("TranslateMessage")
	procDispatchMessageW  = user32.NewProc("DispatchMessageW")
	procBeginPaint        = user32.NewProc("BeginPaint")
	procEndPaint          = user32.NewProc("EndPaint")
	procGetClientRect     = user32.NewProc("GetClientRect")
	procInvalidateRect    = user32.NewProc("InvalidateRect")
	procLoadCursorW       = user32.NewProc("LoadCursorW")
	procScreenToClient    = user32.NewProc("ScreenToClient")
	procGetWindowRect     = user32.NewProc("GetWindowRect")
	procStretchDIBits     = gdi32.NewProc("StretchDIBits")
	procSetDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
	procPatBlt            = gdi32.NewProc("PatBlt")
	procDwmFlush          = dwmapi.NewProc("DwmFlush")
	procCreateDIBSection  = gdi32.NewProc("CreateDIBSection")
	procCreateCompatDC    = gdi32.NewProc("CreateCompatibleDC")
	procSelectObject      = gdi32.NewProc("SelectObject")
	procBitBlt            = gdi32.NewProc("BitBlt")
	procStretchBlt        = gdi32.NewProc("StretchBlt")
	procSetStretchBltMode = gdi32.NewProc("SetStretchBltMode")
	procDeleteObject      = gdi32.NewProc("DeleteObject")
	procDeleteDC          = gdi32.NewProc("DeleteDC")
	procGetModuleHandleW  = kernel32.NewProc("GetModuleHandleW")
)

type point struct {
	X, Y int32
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type msg struct {
	HWND    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type paintStruct struct {
	HDC         uintptr
	FErase      int32
	RCPaint     rect
	FRestore    int32
	FIncUpdate  int32
	RGBReserved [32]byte
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors uint32
}

var (
	proxyMu     sync.Mutex
	proxyByHWND = map[uintptr]*proxyWindow{}
	classOnce   sync.Once
	classErr    error
	wndProcFn   = windows.NewCallback(wndProc)
)

type proxyWindow struct {
	session *Session
	preview *previewWindow
	hwnd    uintptr
	ready   chan struct{}
	done    chan struct{}
	initErr error

	frameMu sync.RWMutex
	frame   []byte
	frameW  int
	frameH  int

	clientMu sync.RWMutex
	clientW  int
	clientH  int

	firstOnce sync.Once
	closeOnce sync.Once

	paintCalls  atomic.Int64
	printCalls  atomic.Int64
	lastStretch atomic.Int64
	paintEmpty  atomic.Int64
	mouseDown   atomic.Int64
	mouseUp     atomic.Int64
	mouseMove   atomic.Int64

	memDC   uintptr
	dib     uintptr
	dibBits unsafe.Pointer
	dibW    int
	dibH    int

	pendingDown bool
	dragging    bool
	downX       int
	downY       int
}

func newProxyWindow(s *Session) (*proxyWindow, error) {
	w := &proxyWindow{
		session: s,
		ready:   make(chan struct{}),
		done:    make(chan struct{}),
	}
	go w.run()
	<-w.ready
	if w.initErr != nil {
		return nil, w.initErr
	}
	return w, nil
}

func (w *proxyWindow) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	classOnce.Do(func() {
		className, _ := windows.UTF16PtrFromString("MaaGORDProxyWindow")
		hInst, _, _ := procGetModuleHandleW.Call(0)
		cursor, _, _ := procLoadCursorW.Call(0, 32512) // IDC_ARROW
		wc := wndClassEx{
			CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
			Style:         0x0003, // CS_HREDRAW | CS_VREDRAW
			LpfnWndProc:   wndProcFn,
			HInstance:     hInst,
			HCursor:       cursor,
			LpszClassName: className,
		}
		atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		if atom == 0 {
			classErr = fmt.Errorf("RegisterClassExW: %v", err)
		}
	})
	if classErr != nil {
		w.initErr = classErr
		close(w.ready)
		return
	}

	title, _ := windows.UTF16PtrFromString("MaaGO RDP capture")
	className, _ := windows.UTF16PtrFromString("MaaGORDProxyWindow")
	hInst, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		0x80000000, // WS_POPUP, hidden; MaaCore captures it through the hook
		0, 0,
		uintptr(w.session.cfg.Width),
		uintptr(w.session.cfg.Height),
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		w.initErr = fmt.Errorf("CreateWindowExW: %v", err)
		close(w.ready)
		return
	}
	w.hwnd = hwnd
	proxyMu.Lock()
	proxyByHWND[hwnd] = w
	proxyMu.Unlock()
	w.updateClientSize()
	close(w.ready)

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

func lookupProxy(hwnd uintptr) *proxyWindow {
	proxyMu.Lock()
	defer proxyMu.Unlock()
	return proxyByHWND[hwnd]
}

func (w *proxyWindow) HWND() uintptr { return w.hwnd }

func (w *proxyWindow) Show() {
	if w.hwnd == 0 {
		return
	}
	w.positionOnScreen()
	procShowWindow.Call(w.hwnd, 9) // SW_RESTORE
	procSetForeground.Call(w.hwnd)
}

// positionOnScreen keeps the proxy window composed by DWM. A window created
// with CW_USEDEFAULT can end up above the visible desktop; PrintWindow with
// PW_RENDERFULLCONTENT then captures a black surface.
func (w *proxyWindow) positionOnScreen() {
	if w.hwnd == 0 {
		return
	}
	ret, _, _ := procSetWindowPos.Call(
		w.hwnd,
		^uintptr(0), // HWND_TOPMOST
		0, 0,
		uintptr(w.session.cfg.Width+16),
		uintptr(w.session.cfg.Height+39),
		0x0040|0x0010, // SWP_SHOWWINDOW | SWP_NOACTIVATE
	)
	procDwmFlush.Call()
	if w.paintCalls.Load() <= 3 {
		log.Printf("rdp proxy position ret=%d", ret)
	}
}

func (w *proxyWindow) Close() {
	w.closeOnce.Do(func() {
		if w.hwnd != 0 {
			procPostMessageW.Call(w.hwnd, wmProxyDestroy, 0, 0)
		}
	})
	select {
	case <-w.done:
	case <-time.After(2 * time.Second):
	}
}

func (w *proxyWindow) SetFrame(img *image.RGBA) {
	if img == nil {
		return
	}
	b := img.Bounds()
	fw, fh := b.Dx(), b.Dy()
	if fw <= 0 || fh <= 0 {
		return
	}
	need := fw * fh * 4
	w.frameMu.Lock()
	if cap(w.frame) < need {
		w.frame = make([]byte, need)
	} else {
		w.frame = w.frame[:need]
	}
	for y := 0; y < fh; y++ {
		src := img.Pix[y*img.Stride : y*img.Stride+fw*4]
		dst := w.frame[y*fw*4 : (y+1)*fw*4]
		for x := 0; x < fw; x++ {
			dst[x*4] = src[x*4+2]
			dst[x*4+1] = src[x*4+1]
			dst[x*4+2] = src[x*4]
			dst[x*4+3] = 255
		}
	}
	w.frameW, w.frameH = fw, fh
	w.frameMu.Unlock()
	if w.preview != nil {
		w.preview.invalidate()
	}
	w.firstOnce.Do(func() { close(w.session.firstFrame) })
}

func (w *proxyWindow) updateClientSize() {
	if w.hwnd == 0 {
		return
	}
	var r rect
	ok, _, _ := procGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&r)))
	if ok == 0 {
		return
	}
	w.clientMu.Lock()
	w.clientW = int(r.Right - r.Left)
	w.clientH = int(r.Bottom - r.Top)
	w.clientMu.Unlock()
}

func (w *proxyWindow) paint(hdc uintptr) {
	if hdc == 0 {
		return
	}
	w.paintCalls.Add(1)
	w.frameMu.RLock()
	if len(w.frame) == 0 || w.frameW <= 0 || w.frameH <= 0 {
		w.paintEmpty.Add(1)
		w.frameMu.RUnlock()
		return
	}
	fw, fh := w.frameW, w.frameH
	w.clientMu.RLock()
	cw, ch := w.clientW, w.clientH
	w.clientMu.RUnlock()
	if cw <= 0 || ch <= 0 {
		cw, ch = fw, fh
	}

	if w.memDC == 0 || w.dibW != fw || w.dibH != fh {
		if w.dib != 0 {
			procDeleteObject.Call(w.dib)
		}
		if w.memDC != 0 {
			procDeleteDC.Call(w.memDC)
		}
		w.memDC, w.dib, w.dibBits, w.dibW, w.dibH = 0, 0, nil, 0, 0
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
		dib, _, _ := procCreateDIBSection.Call(
			hdc,
			uintptr(unsafe.Pointer(&bi)),
			0,
			uintptr(unsafe.Pointer(&bits)),
			0,
			0,
		)
		if mem != 0 && dib != 0 && bits != nil {
			w.memDC, w.dib, w.dibBits, w.dibW, w.dibH = mem, dib, bits, fw, fh
			procSelectObject.Call(mem, dib)
			if w.paintCalls.Load() <= 3 {
				log.Printf("rdp proxy dib: %dx%d mem=%x dib=%x", fw, fh, mem, dib)
			}
		}
	}

	if w.memDC != 0 && w.dibBits != nil {
		dst := unsafe.Slice((*byte)(w.dibBits), fw*fh*4)
		copy(dst, w.frame)
		r, _, _ := procBitBlt.Call(
			hdc,
			0, 0, uintptr(cw), uintptr(ch),
			w.memDC,
			0, 0,
			0x00CC0020, // SRCCOPY
		)
		w.lastStretch.Store(int64(r))
		if w.paintCalls.Load() <= 3 {
			log.Printf("rdp proxy bitblt: ret=%d client=%dx%d", r, cw, ch)
		}
	}
	w.frameMu.RUnlock()
}

func (w *proxyWindow) Stats() (paint, prints, stretch, empty, down, up, move int64) {
	return w.paintCalls.Load(), w.printCalls.Load(), w.lastStretch.Load(), w.paintEmpty.Load(),
		w.mouseDown.Load(), w.mouseUp.Load(), w.mouseMove.Load()
}

func (w *proxyWindow) scale(x, y int) (uint16, uint16) {
	w.clientMu.RLock()
	cw, ch := w.clientW, w.clientH
	w.clientMu.RUnlock()
	if cw <= 0 || ch <= 0 {
		return uint16(x), uint16(y)
	}
	rw, rh := w.session.cfg.Width, w.session.cfg.Height
	return uint16(x * rw / cw), uint16(y * rh / ch)
}

func clientXY(lParam uintptr) (int, int) {
	x := int(int16(lParam & 0xffff))
	y := int(int16((lParam >> 16) & 0xffff))
	return x, y
}

func (w *proxyWindow) onMouseMove(wParam, lParam uintptr) {
	if w.session == nil {
		return
	}
	w.mouseMove.Add(1)
	x, y := clientXY(lParam)
	sx, sy := w.scale(x, y)
	if w.pendingDown && wParam&0x0001 != 0 { // MK_LBUTTON
		if !w.dragging {
			w.session.mouseButton(gordp.ButtonLeft, true, uint16(w.downX), uint16(w.downY))
			w.dragging = true
		}
		w.session.mouseButton(gordp.ButtonLeft, true, sx, sy)
		return
	}
	w.session.mouseMove(sx, sy)
}

func (w *proxyWindow) onLeftDown(lParam uintptr) {
	x, y := clientXY(lParam)
	sx, sy := w.scale(x, y)
	w.pendingDown = true
	w.dragging = false
	w.downX, w.downY = int(sx), int(sy)
	if n := w.mouseDown.Add(1); n <= 3 {
		log.Printf("rdp proxy mouse down client=%d,%d remote=%d,%d", x, y, sx, sy)
	}
}

func (w *proxyWindow) onLeftUp(lParam uintptr) {
	if !w.pendingDown {
		return
	}
	x, y := clientXY(lParam)
	sx, sy := w.scale(x, y)
	if !w.dragging {
		// Use the same fast-path sequence as a normal RDP click: move, down,
		// up in one write. Separate down/up writes were ignored by the game.
		w.session.click(uint16(w.downX), uint16(w.downY))
	} else {
		w.session.mouseButton(gordp.ButtonLeft, false, sx, sy)
	}
	w.pendingDown = false
	w.dragging = false
	if n := w.mouseUp.Add(1); n <= 3 {
		log.Printf("rdp proxy mouse up client=%d,%d remote=%d,%d", x, y, sx, sy)
	}
}

func (w *proxyWindow) onRightButton(down bool, lParam uintptr) {
	if w.session == nil {
		return
	}
	x, y := clientXY(lParam)
	sx, sy := w.scale(x, y)
	w.session.mouseButton(gordp.ButtonRight, down, sx, sy)
	if down {
		w.mouseDown.Add(1)
	} else {
		w.mouseUp.Add(1)
	}
}

func (w *proxyWindow) onWheel(wParam, lParam uintptr) {
	delta := int16(wParam >> 16)
	amount := int16(delta / 120)
	if amount == 0 {
		if delta > 0 {
			amount = 1
		} else {
			amount = -1
		}
	}
	x := int(int16(lParam & 0xffff))
	y := int(int16((lParam >> 16) & 0xffff))
	var pt point
	pt.X, pt.Y = int32(x), int32(y)
	procScreenToClient.Call(w.hwnd, uintptr(unsafe.Pointer(&pt)))
	sx, sy := w.scale(int(pt.X), int(pt.Y))
	w.session.mouseWheel(sx, sy, amount)
}

func (w *proxyWindow) onKey(msg uint32, wParam, lParam uintptr) {
	down := msg == wmKeyDown || msg == wmSysKeyDown
	scan := byte((lParam >> 16) & 0xff)
	ext := (lParam>>24)&1 != 0
	if scan == 0 {
		return
	}
	w.session.key(scan, down, ext)
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	w := lookupProxy(hwnd)
	switch uint32(message) {
	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		if w != nil {
			w.paint(hdc)
		}
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmPrint, wmPrintClient:
		if w != nil {
			w.printCalls.Add(1)
			w.paint(wParam)
		}
		return 0
	case wmEraseBkgnd:
		return 1
	case wmSize:
		if w != nil {
			w.updateClientSize()
		}
		return 0
	case wmMouseMove:
		if w != nil {
			w.onMouseMove(wParam, lParam)
		}
		return 0
	case wmLButtonDown:
		if w != nil {
			w.onLeftDown(lParam)
		}
		return 0
	case wmLButtonUp:
		if w != nil {
			w.onLeftUp(lParam)
		}
		return 0
	case wmRButtonDown:
		if w != nil {
			w.onRightButton(true, lParam)
		}
		return 0
	case wmRButtonUp:
		if w != nil {
			w.onRightButton(false, lParam)
		}
		return 0
	case wmMouseWheel:
		if w != nil {
			w.onWheel(wParam, lParam)
		}
		return 0
	case wmKeyDown, wmKeyUp, wmSysKeyDown, wmSysKeyUp:
		if w != nil {
			w.onKey(uint32(message), wParam, lParam)
		}
		return 0
	case wmChar:
		if w != nil {
			w.session.char(rune(wParam))
		}
		return 0
	case wmClose:
		// Closing the preview must not break MaaCore's capture. Hide it; the
		// PrintWindow hook still serves frames from the buffer.
		procShowWindow.Call(hwnd, 0) // SW_HIDE
		return 0
	case wmProxyDestroy:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		proxyMu.Lock()
		delete(proxyByHWND, hwnd)
		proxyMu.Unlock()
		if w != nil {
			if w.dib != 0 {
				procDeleteObject.Call(w.dib)
			}
			if w.memDC != 0 {
				procDeleteDC.Call(w.memDC)
			}
			select {
			case <-w.done:
			default:
				close(w.done)
			}
		}
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}
