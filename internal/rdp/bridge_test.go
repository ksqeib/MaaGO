package rdp

import (
	"encoding/binary"
	"image"
	"io"
	"net"
	"testing"
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
	if w != 2 || h != 2 || size != 16 {
		t.Fatalf("unexpected frame header: %dx%d size=%d", w, h, size)
	}
	pixels := make([]byte, size)
	if _, err := io.ReadFull(conn, pixels); err != nil {
		t.Fatal(err)
	}
	// RGBA 0x40,0x40,0x40,0x40 becomes BGRA with an opaque alpha.
	if pixels[0] != 0x40 || pixels[3] != 0xFF {
		t.Fatalf("unexpected BGRA pixel: % X", pixels[:4])
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
