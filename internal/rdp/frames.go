package rdp

import (
	"image"
	"math"
	"sync"
)

// frameStore keeps the latest remote frame as 3-channel BGR. MaaCore's Win32
// control-unit ABI expects BGR (the official screencaps call bgra_to_bgr()),
// and its OCR path silently returns empty results when given 4-channel BGRA.
type frameStore struct {
	mu   sync.RWMutex
	data []byte
	w    int
	h    int
	seq  uint64

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
	need := fw * fh * 3
	f.mu.Lock()
	if cap(f.data) < need {
		f.data = make([]byte, need)
	} else {
		f.data = f.data[:need]
	}
	for y := 0; y < fh; y++ {
		src := img.Pix[y*img.Stride : y*img.Stride+fw*4]
		dst := f.data[y*fw*3 : (y+1)*fw*3]
		for x := 0; x < fw; x++ {
			dst[x*3] = src[x*4+2]
			dst[x*3+1] = src[x*4+1]
			dst[x*3+2] = src[x*4]
		}
	}
	f.w, f.h = fw, fh
	f.seq++
	f.mu.Unlock()
	f.firstOnce.Do(func() { close(f.first) })
}

// version increases whenever a new remote frame has been applied.
func (f *frameStore) version() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.seq
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
			i := (y*w + x) * 3
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
