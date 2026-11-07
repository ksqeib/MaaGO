package rdp

import (
	"image"
	"math"
	"sync"
)

// frameStore keeps the latest remote frame in BGRA order, which is what the
// Win32 control-unit ABI expects from a 32-bit DIB. It replaces the old hidden
// proxy window: MaaCore no longer needs an HWND, and the frame never has to be
// painted onto the local desktop.
type frameStore struct {
	mu   sync.RWMutex
	data []byte
	w    int
	h    int

	firstOnce sync.Once
	first     chan struct{}
}

func newFrameStore() *frameStore {
	return &frameStore{first: make(chan struct{})}
}

func (f *frameStore) set(img *image.RGBA) {
	if img == nil {
		return
	}
	b := img.Bounds()
	fw, fh := b.Dx(), b.Dy()
	if fw <= 0 || fh <= 0 {
		return
	}
	need := fw * fh * 4
	f.mu.Lock()
	if cap(f.data) < need {
		f.data = make([]byte, need)
	} else {
		f.data = f.data[:need]
	}
	for y := 0; y < fh; y++ {
		src := img.Pix[y*img.Stride : y*img.Stride+fw*4]
		dst := f.data[y*fw*4 : (y+1)*fw*4]
		for x := 0; x < fw; x++ {
			dst[x*4] = src[x*4+2]
			dst[x*4+1] = src[x*4+1]
			dst[x*4+2] = src[x*4]
			dst[x*4+3] = 255
		}
	}
	f.w, f.h = fw, fh
	f.mu.Unlock()
	f.firstOnce.Do(func() { close(f.first) })
}

func (f *frameStore) hasFrame() bool {
	select {
	case <-f.first:
		return true
	default:
		return false
	}
}

// snapshot returns a private copy so the caller can stream it without holding
// the frame lock while the RDP pump writes the next frame.
func (f *frameStore) snapshot() ([]byte, int, int) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	if len(f.data) == 0 || f.w <= 0 || f.h <= 0 {
		return nil, 0, 0
	}
	out := make([]byte, len(f.data))
	copy(out, f.data)
	return out, f.w, f.h
}

func (f *frameStore) standardDeviation() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	w, h := f.w, f.h
	if len(f.data) == 0 || w <= 0 || h <= 0 {
		return 0
	}
	stepX := w / 80
	stepY := h / 45
	if stepX < 1 {
		stepX = 1
	}
	if stepY < 1 {
		stepY = 1
	}
	var sum, sum2 float64
	var count int
	for y := 0; y < h; y += stepY {
		for x := 0; x < w; x += stepX {
			i := (y*w + x) * 4
			b := float64(f.data[i])
			g := float64(f.data[i+1])
			r := float64(f.data[i+2])
			lum := 0.299*r + 0.587*g + 0.114*b
			sum += lum
			sum2 += lum * lum
			count++
		}
	}
	if count == 0 {
		return 0
	}
	mean := sum / float64(count)
	variance := sum2/float64(count) - mean*mean
	if variance <= 0 {
		return 0
	}
	return math.Sqrt(variance)
}
