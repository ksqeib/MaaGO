package rdp

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/casvisor/gordp"
)

// The custom MaaWin32ControlUnit.dll talks to this loopback server instead of
// owning a local proxy window. Commands are tiny; screencap is the only large
// payload (one BGRA frame per capture).
const (
	opConnect byte = iota + 1
	opConnected
	opUUID
	opFeatures
	opScreencap
	opClick
	opSwipe
	opTouchDown
	opTouchMove
	opTouchUp
	opKeyDown
	opKeyUp
	opClickKey
	opInputText
	opInactive
	opStartApp
	opStopApp
)

// Bridge serves one MaaCore control unit over loopback TCP.
type Bridge struct {
	listener net.Listener
	session  *Session

	mu   sync.Mutex
	conn net.Conn
	done chan struct{}
	once sync.Once
}

// StartBridge starts the loopback control-unit bridge for a live RDP session.
func StartBridge(session *Session) (*Bridge, error) {
	if session == nil {
		return nil, fmt.Errorf("RDP 会话为空")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("启动 RDP 控制桥失败：%w", err)
	}
	b := &Bridge{
		listener: ln,
		session:  session,
		done:     make(chan struct{}),
	}
	go b.acceptLoop()
	return b, nil
}

// Addr is the host:port the control-unit DLL should connect to.
func (b *Bridge) Addr() string {
	if b == nil || b.listener == nil {
		return ""
	}
	return b.listener.Addr().String()
}

// Close stops accepting connections and drops the active one.
func (b *Bridge) Close() {
	if b == nil {
		return
	}
	b.once.Do(func() {
		close(b.done)
		if b.listener != nil {
			_ = b.listener.Close()
		}
		b.mu.Lock()
		if b.conn != nil {
			_ = b.conn.Close()
		}
		b.mu.Unlock()
	})
}

func (b *Bridge) acceptLoop() {
	for {
		conn, err := b.listener.Accept()
		if err != nil {
			select {
			case <-b.done:
				return
			default:
				continue
			}
		}
		b.mu.Lock()
		old := b.conn
		b.conn = conn
		b.mu.Unlock()
		if old != nil {
			_ = old.Close()
		}
		go b.serve(conn)
	}
}

type bridgeInputState struct {
	down  bool
	lastX uint16
	lastY uint16
}

func (b *Bridge) serve(conn net.Conn) {
	defer conn.Close()
	state := &bridgeInputState{}
	var one [1]byte
	for {
		if _, err := io.ReadFull(conn, one[:]); err != nil {
			return
		}
		if err := b.handle(conn, one[0], state); err != nil {
			return
		}
	}
}

func (b *Bridge) handle(conn net.Conn, op byte, state *bridgeInputState) error {
	switch op {
	case opConnect:
		return writeBool(conn, b.session.Alive())
	case opConnected:
		return writeBool(conn, b.session.Alive())
	case opUUID:
		return writeString(conn, b.session.Config().Key())
	case opFeatures:
		// UseMouseDownAndUpInsteadOfClick | UseKeyboardDownAndUpInsteadOfClick
		return writeUint64(conn, 3)
	case opScreencap:
		data, w, h := b.session.frames.snapshot()
		if data == nil {
			return writeBool(conn, false)
		}
		if err := writeBool(conn, true); err != nil {
			return err
		}
		var header [12]byte
		binary.BigEndian.PutUint32(header[0:4], uint32(w))
		binary.BigEndian.PutUint32(header[4:8], uint32(h))
		binary.BigEndian.PutUint32(header[8:12], uint32(len(data)))
		if _, err := conn.Write(header[:]); err != nil {
			return err
		}
		_, err := conn.Write(data)
		return err
	case opClick:
		x, y, err := readPoint(conn)
		if err != nil {
			return err
		}
		b.session.click(x, y)
		return writeBool(conn, true)
	case opSwipe:
		x1, y1, x2, y2, duration, err := readSwipe(conn)
		if err != nil {
			return err
		}
		b.session.swipe(x1, y1, x2, y2, duration)
		return writeBool(conn, true)
	case opTouchDown:
		x, y, err := readPoint(conn)
		if err != nil {
			return err
		}
		state.down = true
		state.lastX, state.lastY = x, y
		b.session.mouseButton(gordp.ButtonLeft, true, x, y)
		return writeBool(conn, true)
	case opTouchMove:
		x, y, err := readPoint(conn)
		if err != nil {
			return err
		}
		state.lastX, state.lastY = x, y
		if state.down {
			b.session.mouseButton(gordp.ButtonLeft, true, x, y)
		} else {
			b.session.mouseMove(x, y)
		}
		return writeBool(conn, true)
	case opTouchUp:
		if _, err := readUint32(conn); err != nil {
			return err
		}
		if state.down {
			b.session.mouseButton(gordp.ButtonLeft, false, state.lastX, state.lastY)
			state.down = false
		}
		return writeBool(conn, true)
	case opKeyDown:
		vk, err := readUint32(conn)
		if err != nil {
			return err
		}
		code, ext, ok := vkScanCode(int32(vk))
		if ok {
			b.session.key(code, true, ext)
		}
		return writeBool(conn, ok)
	case opKeyUp:
		vk, err := readUint32(conn)
		if err != nil {
			return err
		}
		code, ext, ok := vkScanCode(int32(vk))
		if ok {
			b.session.key(code, false, ext)
		}
		return writeBool(conn, ok)
	case opClickKey:
		vk, err := readUint32(conn)
		if err != nil {
			return err
		}
		code, ext, ok := vkScanCode(int32(vk))
		if ok {
			b.session.key(code, true, ext)
			time.Sleep(50 * time.Millisecond)
			b.session.key(code, false, ext)
		}
		return writeBool(conn, ok)
	case opInputText:
		text, err := readString(conn)
		if err != nil {
			return err
		}
		b.session.typeText(text)
		return writeBool(conn, true)
	case opInactive:
		return writeBool(conn, true)
	case opStartApp, opStopApp:
		if _, err := readString(conn); err != nil {
			return err
		}
		return writeBool(conn, true)
	default:
		return fmt.Errorf("unknown RDP bridge op %d", op)
	}
}

func writeBool(w io.Writer, value bool) error {
	var b [1]byte
	if value {
		b[0] = 1
	}
	_, err := w.Write(b[:])
	return err
}

func writeUint64(w io.Writer, value uint64) error {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], value)
	_, err := w.Write(b[:])
	return err
}

func writeString(w io.Writer, value string) error {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(len(value)))
	if _, err := w.Write(b[:]); err != nil {
		return err
	}
	_, err := io.WriteString(w, value)
	return err
}

func readUint32(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b[:]), nil
}

func readPoint(r io.Reader) (uint16, uint16, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, 0, err
	}
	return uint16(binary.BigEndian.Uint32(b[0:4])), uint16(binary.BigEndian.Uint32(b[4:8])), nil
}

func readSwipe(r io.Reader) (uint16, uint16, uint16, uint16, int, error) {
	var b [20]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, 0, 0, 0, 0, err
	}
	return uint16(binary.BigEndian.Uint32(b[0:4])),
		uint16(binary.BigEndian.Uint32(b[4:8])),
		uint16(binary.BigEndian.Uint32(b[8:12])),
		uint16(binary.BigEndian.Uint32(b[12:16])),
		int(binary.BigEndian.Uint32(b[16:20])), nil
}

func readString(r io.Reader) (string, error) {
	n, err := readUint32(r)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	if n > 1<<20 {
		return "", fmt.Errorf("RDP bridge string too large: %d", n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (s *Session) swipe(x1, y1, x2, y2 uint16, duration int) {
	if duration <= 0 {
		duration = 300
	}
	const steps = 12
	s.mouseButton(gordp.ButtonLeft, true, x1, y1)
	for i := 1; i <= steps; i++ {
		x := uint16(int(x1) + (int(x2)-int(x1))*i/steps)
		y := uint16(int(y1) + (int(y2)-int(y1))*i/steps)
		s.mouseButton(gordp.ButtonLeft, true, x, y)
		time.Sleep(time.Duration(duration/steps) * time.Millisecond)
	}
	s.mouseButton(gordp.ButtonLeft, false, x2, y2)
}
