// Package rdp keeps an outbound RDP session to a Windows machine and exposes
// its desktop as a local window. MaaCore binds to that window with its normal
// Win32 controller, so the task engine runs locally while the game runs on the
// remote machine.
package rdp

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/casvisor/gordp"
)

// Config describes the remote Windows host.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Domain   string
	Width    int
	Height   int
}

func (c Config) normalized() Config {
	if c.Port == 0 {
		c.Port = 3389
	}
	if c.Width == 0 {
		c.Width = 1280
	}
	if c.Height == 0 {
		c.Height = 720
	}
	c.Host = strings.TrimSpace(c.Host)
	c.Username = strings.TrimSpace(c.Username)
	c.Domain = strings.TrimSpace(c.Domain)
	return c
}

func (c Config) Key() string {
	c = c.normalized()
	return fmt.Sprintf("%s:%d/%s/%dx%d", c.Host, c.Port, c.Username, c.Width, c.Height)
}

// Session owns one RDP connection and its local proxy window.
type Session struct {
	cfg     Config
	client  *gordp.Client
	win     *proxyWindow
	preview *previewWindow

	writeMu sync.Mutex
	closed  chan struct{}
	once    sync.Once

	firstFrame chan struct{}
	frameOnce  sync.Once

	errMu   sync.Mutex
	lastErr error

	pointerDirty atomic.Bool
}

// Open authenticates to the remote host and starts pumping frames into a proxy
// window. The returned session stays connected until Close.
func Open(cfg Config) (*Session, error) {
	cfg = cfg.normalized()
	if cfg.Host == "" {
		return nil, fmt.Errorf("RDP 主机为空")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("RDP 用户名为空")
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	client := gordp.NewClient(
		addr,
		cfg.Domain,
		cfg.Username,
		cfg.Password,
		gordp.WithDesktopSize(uint16(cfg.Width), uint16(cfg.Height)),
		gordp.WithTimeout(20*time.Second),
	)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("RDP 连接/认证失败：%w", err)
	}
	s := &Session{
		cfg:        cfg,
		client:     client,
		closed:     make(chan struct{}),
		firstFrame: make(chan struct{}),
	}
	win, err := newProxyWindow(s)
	if err != nil {
		_ = client.Close()
		return nil, err
	}
	s.win = win
	go s.pump()
	select {
	case <-s.firstFrame:
	case <-time.After(15 * time.Second):
		// Keep the session: some desktops only send the first full frame after
		// an input event. Diagnostics reports whether a frame arrived.
	}
	return s, nil
}

func (s *Session) pump() {
	for {
		select {
		case <-s.closed:
			return
		default:
		}
		if _, err := func() (int, error) {
			// gordp writes fast-path input directly to the TLS connection. Serialize
			// it with the pump so input and frame acknowledgements cannot interleave.
			s.writeMu.Lock()
			defer s.writeMu.Unlock()
			return s.client.PumpUpdates(50 * time.Millisecond)
		}(); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "timeout") {
				continue
			}
			s.fail(err)
			return
		}
		if fb := s.client.Framebuffer(); fb != nil {
			s.win.SetFrame(fb)
		}
		select {
		case <-s.closed:
			return
		default:
		}
	}
}

func (s *Session) fail(err error) {
	s.errMu.Lock()
	if s.lastErr == nil {
		s.lastErr = err
	}
	s.errMu.Unlock()
}

func (s *Session) Alive() bool {
	select {
	case <-s.closed:
		return false
	default:
	}
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.lastErr == nil
}

func (s *Session) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.lastErr
}

// HWND is the local proxy window that MaaCore attaches to.
func (s *Session) HWND() uintptr {
	if s.win == nil {
		return 0
	}
	return s.win.HWND()
}

// HasFrame reports whether at least one remote frame has been painted.
func (s *Session) HasFrame() bool {
	select {
	case <-s.firstFrame:
		return true
	default:
		return false
	}
}

func (s *Session) Config() Config { return s.cfg }

// Stats reports proxy paint/PrintWindow/input counters for diagnostics.
func (s *Session) Stats() (paint, prints, stretch, empty, down, up, move int64) {
	if s.win == nil {
		return 0, 0, 0, 0, 0, 0, 0
	}
	return s.win.Stats()
}

// DesktopSize is the remote desktop size requested from the server.
func (s *Session) DesktopSize() (int, int) { return s.cfg.Width, s.cfg.Height }

// Show brings the local preview window to the foreground.
func (s *Session) Show() {
	if s.preview != nil {
		s.preview.Show()
	}
}

// EnsureVisible repositions the small preview window without taking focus.
func (s *Session) EnsureVisible() {
	if s.preview != nil {
		s.preview.position()
	}
}

// SaveFrame writes the last converted frame as a PNG. It is a debugging aid for
// the proxy pipeline and for diagnosing a black or white capture.
func (s *Session) SaveFrame(path string) error {
	if s.win == nil {
		return fmt.Errorf("window is not ready")
	}
	s.win.frameMu.RLock()
	defer s.win.frameMu.RUnlock()
	if len(s.win.frame) == 0 || s.win.frameW <= 0 || s.win.frameH <= 0 {
		return fmt.Errorf("no frame received yet")
	}
	img := image.NewRGBA(image.Rect(0, 0, s.win.frameW, s.win.frameH))
	for y := 0; y < s.win.frameH; y++ {
		src := s.win.frame[y*s.win.frameW*4 : (y+1)*s.win.frameW*4]
		dst := img.Pix[y*img.Stride : y*img.Stride+s.win.frameW*4]
		for x := 0; x < s.win.frameW; x++ {
			dst[x*4] = src[x*4+2]
			dst[x*4+1] = src[x*4+1]
			dst[x*4+2] = src[x*4]
			dst[x*4+3] = 255
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// FrameStd returns the luminance standard deviation of the latest frame. A
// locked or powered-off desktop is nearly uniform; a game screen is not.
func (s *Session) FrameStd() float64 {
	if s.win == nil {
		return 0
	}
	s.win.frameMu.RLock()
	defer s.win.frameMu.RUnlock()
	w, h := s.win.frameW, s.win.frameH
	if len(s.win.frame) == 0 || w <= 0 || h <= 0 {
		return 0
	}
	stepX := w / 80
	if stepX < 1 {
		stepX = 1
	}
	stepY := h / 45
	if stepY < 1 {
		stepY = 1
	}
	var sum, sum2 float64
	var n float64
	for y := 0; y < h; y += stepY {
		for x := 0; x < w; x += stepX {
			i := (y*w + x) * 4
			b := float64(s.win.frame[i])
			g := float64(s.win.frame[i+1])
			r := float64(s.win.frame[i+2])
			lum := 0.299*r + 0.587*g + 0.114*b
			sum += lum
			sum2 += lum * lum
			n++
		}
	}
	if n == 0 {
		return 0
	}
	mean := sum / n
	variance := sum2/n - mean*mean
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}

// LooksLocked reports whether the latest frame looks like a lock screen or a
// blank/off display.
func (s *Session) LooksLocked() bool {
	return s.FrameStd() < 22
}

func (s *Session) pressEnter() {
	s.key(0x1C, true, false)
	s.key(0x1C, false, false)
}

func (s *Session) typeText(text string) {
	if text == "" {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.client.TypeUnicode(text)
}

// UnlockIfLocked dismisses the Windows lock screen after an RDP reconnect. The
// RDP client has already authenticated via NLA; usually Enter is enough, with a
// password fallback for hosts that show a credential prompt.
func (s *Session) UnlockIfLocked(password string) bool {
	if !s.LooksLocked() {
		return false
	}
	s.pressEnter()
	time.Sleep(1200 * time.Millisecond)
	if !s.LooksLocked() {
		return true
	}
	s.typeText(password)
	time.Sleep(200 * time.Millisecond)
	s.pressEnter()
	time.Sleep(1200 * time.Millisecond)
	return true
}

// FocusRemote clicks a harmless edge of the remote desktop so the game window
// becomes active after an RDP reconnect. Without this the first click can be
// consumed by window activation.
func (s *Session) FocusRemote() {
	for i := 0; i < 2; i++ {
		s.click(10, 450)
		time.Sleep(200 * time.Millisecond)
	}
	s.CenterPointer()
}

// pointerSettle is how long the PC client needs to finish its mouse-position
// parallax animation before a screenshot is taken.
const pointerSettle = 90 * time.Millisecond

// CenterPointer parks the remote pointer at the middle of the game window.
// The PC client tilts the UI based on the pointer position; a centered pointer
// keeps the parallax neutral. It is also far away from most home-screen cards.
func (s *Session) CenterPointer() {
	if s == nil || s.client == nil {
		return
	}
	s.centerPointer()
	s.pointerDirty.Store(false)
}

func (s *Session) centerPointer() {
	x := uint16(s.cfg.Width / 2)
	y := uint16(s.cfg.Height / 2)
	s.mouseMove(x, y)
	time.Sleep(pointerSettle)
}

func (s *Session) markPointerDirty() {
	if s != nil {
		s.pointerDirty.Store(true)
	}
}

// ensurePointerParked is called by the PrintWindow hook before MaaCore captures.
// After an automated click the pointer sits on the clicked button; moving it
// back to the center first keeps the next frame's parallax stable.
func (s *Session) ensurePointerParked() {
	if s == nil || !s.pointerDirty.Swap(false) {
		return
	}
	s.centerPointer()
}

// SnapshotJPEG encodes the latest frame, downscaled to maxWidth. It is used by
// the in-app preview; MaaCore itself uses the full-resolution frame.
func (s *Session) SnapshotJPEG(maxWidth, quality int) ([]byte, int, int, error) {
	if s.win == nil {
		return nil, 0, 0, fmt.Errorf("RDP 预览窗口未就绪")
	}
	s.win.frameMu.RLock()
	fw, fh := s.win.frameW, s.win.frameH
	if len(s.win.frame) == 0 || fw <= 0 || fh <= 0 {
		s.win.frameMu.RUnlock()
		return nil, 0, 0, fmt.Errorf("尚未收到远端画面")
	}
	src := make([]byte, fw*fh*4)
	copy(src, s.win.frame)
	s.win.frameMu.RUnlock()

	if maxWidth <= 0 || maxWidth > fw {
		maxWidth = fw
	}
	nw, nh := fw, fh
	if maxWidth < fw {
		nw = maxWidth
		nh = fh * nw / fw
		if nh < 1 {
			nh = 1
		}
	}
	img := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := y * fh / nh
		for x := 0; x < nw; x++ {
			sx := x * fw / nw
			si := (sy*fw + sx) * 4
			di := y*img.Stride + x*4
			img.Pix[di] = src[si+2]
			img.Pix[di+1] = src[si+1]
			img.Pix[di+2] = src[si]
			img.Pix[di+3] = 255
		}
	}
	if quality <= 0 {
		quality = 70
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), nw, nh, nil
}

// ClickAt forwards one left click at remote desktop coordinates.
func (s *Session) ClickAt(x, y int) {
	if x < 0 || y < 0 || x > s.cfg.Width || y > s.cfg.Height {
		return
	}
	s.click(uint16(x), uint16(y))
}

// TypeText forwards text via RDP unicode key events.
func (s *Session) TypeText(text string) { s.typeText(text) }

// WheelAt forwards a mouse wheel event.
func (s *Session) WheelAt(x, y, amount int) {
	if x < 0 || y < 0 || x > s.cfg.Width || y > s.cfg.Height {
		return
	}
	if amount == 0 {
		amount = 1
	}
	s.mouseWheel(uint16(x), uint16(y), int16(amount))
}

var previewKeys = map[string]struct {
	code byte
	ext  bool
}{
	"enter":     {0x1C, false},
	"esc":       {0x01, false},
	"tab":       {0x0F, false},
	"space":     {0x39, false},
	"backspace": {0x0E, false},
	"delete":    {0x53, true},
	"up":        {0x48, true},
	"down":      {0x50, true},
	"left":      {0x4B, true},
	"right":     {0x4D, true},
}

// PressKeyName forwards a named non-text key (enter/esc/tab/...).
func (s *Session) PressKeyName(name string) error {
	key, ok := previewKeys[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return fmt.Errorf("不支持的按键：%s", name)
	}
	s.key(key.code, true, key.ext)
	s.key(key.code, false, key.ext)
	return nil
}

func (s *Session) Close() error {
	var err error
	s.once.Do(func() {
		close(s.closed)
		if s.client != nil {
			err = s.client.Close()
		}
		if s.preview != nil {
			s.preview.Close()
		}
		if s.win != nil {
			s.win.Close()
		}
	})
	return err
}

// Input helpers used by the proxy window. All writes are serialized because
// the RDP client expects one fast-path writer.
func (s *Session) click(x, y uint16) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.client.Click(x, y)
}

func (s *Session) mouseButton(button gordp.MouseButton, down bool, x, y uint16) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if down {
		// RDP only applies coordinates when PTRFLAGS_MOVE is set. Move first on
		// button-down; sending another move before button-up would clear the
		// pressed state, so the up event reuses the down position.
		_ = s.client.MouseMove(x, y)
		_ = s.client.MouseButton(button, true, x, y)
		return
	}
	_ = s.client.MouseButton(button, false, x, y)
}

func (s *Session) mouseMove(x, y uint16) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.client.MouseMove(x, y)
}

func (s *Session) mouseWheel(x, y uint16, amount int16) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.client.MouseWheel(x, y, amount)
}

func (s *Session) key(code byte, down, extended bool) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.client.KeyScancode(code, down, extended)
}

func (s *Session) char(ch rune) {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.client.KeyUnicode(ch)
}

// Manager serializes access to the single RDP session. MaaGO only drives one
// remote machine at a time.
var (
	managerMu  sync.Mutex
	managerCur *Session
	managerKey string
)

// Ensure returns a live session for cfg, reconnecting if the parameters
// changed or the previous connection died.
func Ensure(cfg Config) (*Session, error) {
	cfg = cfg.normalized()
	key := cfg.Key()

	managerMu.Lock()
	defer managerMu.Unlock()
	if managerCur != nil && managerKey == key && managerCur.Alive() {
		return managerCur, nil
	}
	if managerCur != nil {
		_ = managerCur.Close()
		managerCur = nil
		managerKey = ""
	}
	session, err := Open(cfg)
	if err != nil {
		return nil, err
	}
	managerCur = session
	managerKey = key
	return session, nil
}

// Current returns the active session, if any.
func Current() *Session {
	managerMu.Lock()
	defer managerMu.Unlock()
	return managerCur
}

// Shutdown closes the active session.
func Shutdown() {
	managerMu.Lock()
	defer managerMu.Unlock()
	if managerCur != nil {
		_ = managerCur.Close()
		managerCur = nil
		managerKey = ""
	}
}
