package rdp

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"
	"net"
	"testing"

	"github.com/casvisor/gordp"
)

func TestBridgeConnectAndScreencap(t *testing.T) {
	session := &Session{
		frames: newFrameStore(),
		closed: make(chan struct{}),
	}
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for i := range img.Pix {
		img.Pix[i] = 0x40
	}
	session.frames.set(img)

	bridge, err := StartBridge(session)
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()

	conn, err := net.Dial("tcp", bridge.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte{opConnect}); err != nil {
		t.Fatal(err)
	}
	var ok [1]byte
	if _, err := io.ReadFull(conn, ok[:]); err != nil || ok[0] != 1 {
		t.Fatalf("connect response: %v %v", ok[0], err)
	}

	if _, err := conn.Write([]byte{opScreencap}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(conn, ok[:]); err != nil || ok[0] != 1 {
		t.Fatalf("screencap status: %v %v", ok[0], err)
	}
	var header [12]byte
	if _, err := io.ReadFull(conn, header[:]); err != nil {
		t.Fatal(err)
	}
	w := binary.BigEndian.Uint32(header[0:4])
	h := binary.BigEndian.Uint32(header[4:8])
	size := binary.BigEndian.Uint32(header[8:12])
	if w != 2 || h != 2 || size != 12 {
		t.Fatalf("unexpected frame header: %dx%d size=%d", w, h, size)
	}
	pixels := make([]byte, size)
	if _, err := io.ReadFull(conn, pixels); err != nil {
		t.Fatal(err)
	}
	// RGBA 0x40,0x40,0x40,0x40 becomes 3-channel BGR.
	if pixels[0] != 0x40 || pixels[1] != 0x40 || pixels[2] != 0x40 {
		t.Fatalf("unexpected BGR pixel: % X", pixels[:3])
	}
}

type recordedPoint struct {
	x, y uint16
}

type fakeInput struct {
	clicks []recordedPoint
	downs  []recordedPoint
	ups    []recordedPoint
	moves  []recordedPoint
}

func (f *fakeInput) click(x, y uint16) { f.clicks = append(f.clicks, recordedPoint{x, y}) }

func (f *fakeInput) mouseButton(_ gordp.MouseButton, down bool, x, y uint16) {
	if down {
		f.downs = append(f.downs, recordedPoint{x, y})
		return
	}
	f.ups = append(f.ups, recordedPoint{x, y})
}

func (f *fakeInput) mouseMove(x, y uint16) { f.moves = append(f.moves, recordedPoint{x, y}) }

type scriptedConn struct {
	in  *bytes.Buffer
	out bytes.Buffer
}

func (c *scriptedConn) Read(p []byte) (int, error)  { return c.in.Read(p) }
func (c *scriptedConn) Write(p []byte) (int, error) { return c.out.Write(p) }

func pointBytes(x, y uint16) []byte {
	var b [8]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(x))
	binary.BigEndian.PutUint32(b[4:8], uint32(y))
	return b[:]
}

func u32Bytes(value uint32) []byte {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], value)
	return b[:]
}

func runBridgeOp(t *testing.T, bridge *Bridge, state *bridgeInputState, op byte, args []byte) {
	t.Helper()
	conn := &scriptedConn{in: bytes.NewBuffer(args)}
	if err := bridge.handle(conn, op, state); err != nil {
		t.Fatalf("op %d: %v", op, err)
	}
}

// A tap must become one combined fast-path click (move + down + up). Sending
// down and up as separate PDUs is what made the game ignore QuickSwitch taps.
func TestBridgeTapBecomesSingleClick(t *testing.T) {
	input := &fakeInput{}
	bridge := &Bridge{input: input}
	state := &bridgeInputState{}

	runBridgeOp(t, bridge, state, opTouchDown, pointBytes(100, 200))
	if len(input.clicks)+len(input.downs)+len(input.ups) != 0 {
		t.Fatalf("touch_down must not press yet: %+v", input)
	}
	if len(input.moves) != 1 || input.moves[0] != (recordedPoint{100, 200}) {
		t.Fatalf("touch_down should hover at the target: %+v", input.moves)
	}

	runBridgeOp(t, bridge, state, opTouchUp, u32Bytes(0))
	if len(input.clicks) != 1 || input.clicks[0] != (recordedPoint{100, 200}) {
		t.Fatalf("expected one combined click at (100,200), got %+v", input.clicks)
	}
	if len(input.downs) != 0 || len(input.ups) != 0 {
		t.Fatalf("tap should not send separate down/up: %+v", input)
	}
}

func TestBridgeDragUsesDownMoveUp(t *testing.T) {
	input := &fakeInput{}
	bridge := &Bridge{input: input}
	state := &bridgeInputState{}

	runBridgeOp(t, bridge, state, opTouchDown, pointBytes(10, 20))
	runBridgeOp(t, bridge, state, opTouchMove, pointBytes(30, 40))
	runBridgeOp(t, bridge, state, opTouchUp, u32Bytes(0))

	if len(input.clicks) != 0 {
		t.Fatalf("drag must not be sent as a click: %+v", input.clicks)
	}
	if len(input.downs) != 2 || input.downs[0] != (recordedPoint{10, 20}) ||
		input.downs[1] != (recordedPoint{30, 40}) {
		t.Fatalf("unexpected drag downs: %+v", input.downs)
	}
	if len(input.ups) != 1 || input.ups[0] != (recordedPoint{30, 40}) {
		t.Fatalf("unexpected drag ups: %+v", input.ups)
	}
}

func TestBridgeParkingMoveWithoutButton(t *testing.T) {
	input := &fakeInput{}
	bridge := &Bridge{input: input}
	state := &bridgeInputState{}

	runBridgeOp(t, bridge, state, opTouchMove, pointBytes(50, 60))
	if len(input.moves) != 1 || input.moves[0] != (recordedPoint{50, 60}) {
		t.Fatalf("expected a plain pointer move, got %+v", input.moves)
	}
	if len(input.downs)+len(input.ups)+len(input.clicks) != 0 {
		t.Fatalf("parking move must not touch buttons: %+v", input)
	}
}

func TestVKScanCode(t *testing.T) {
	cases := []struct {
		vk   int32
		code byte
		ext  bool
		ok   bool
	}{
		{0x1B, 0x01, false, true}, // Escape
		{0x0D, 0x1C, false, true}, // Enter
		{0x41, 0x1E, false, true}, // A
		{0x25, 0x4B, true, true},  // Left
		{0x2E, 0x53, true, true},  // Delete
		{0x99, 0, false, false},   // unknown
	}
	for _, tc := range cases {
		code, ext, ok := vkScanCode(tc.vk)
		if code != tc.code || ext != tc.ext || ok != tc.ok {
			t.Errorf("vk %#x: got (%#x, %v, %v), want (%#x, %v, %v)",
				tc.vk, code, ext, ok, tc.code, tc.ext, tc.ok)
		}
	}
}
