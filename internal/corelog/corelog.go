// Package corelog writes MaaCore's stdout to its own date-rotated log file.
package corelog

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// Session describes one real-core queue run.
type Session struct {
	Version   string
	Address   string
	ADBPath   string
	TaskNames []string
}

// Writer serialises MaaCore output into logs/core/core-YYYY-MM-DD.log.
type Writer struct {
	mu   sync.Mutex
	dir  string
	date string
	file *os.File
}

// New creates a dated core log writer and prunes files older than 14 days.
func New(dir string) *Writer {
	writer := &Writer{dir: dir}
	_ = os.MkdirAll(dir, 0o755)
	writer.prune(14)
	return writer
}

// Dir returns the log directory.
func (w *Writer) Dir() string {
	if w == nil {
		return ""
	}
	return w.dir
}

// Write implements io.Writer for the stdout pipe. Each call is written as a
// line; trailing CR/LF are normalised.
func (w *Writer) Write(data []byte) (int, error) {
	if w == nil {
		return len(data), nil
	}
	text := strings.TrimRight(string(data), "\r\n")
	if text == "" {
		return len(data), nil
	}
	w.WriteLine(text)
	return len(data), nil
}

// WriteLine appends one line, rotating to a new file when the date changes.
func (w *Writer) WriteLine(line string) {
	if w == nil {
		return
	}
	line = strings.TrimRight(ansiPattern.ReplaceAllString(line, ""), "\r\n")

	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now()
	date := now.Format("2006-01-02")
	if w.file == nil || w.date != date {
		w.openLocked(date)
	}
	if w.file == nil {
		return
	}
	_, _ = w.file.WriteString(line + "\n")
}

// BeginSession writes the visible separator and metadata for one queue run.
func (w *Writer) BeginSession(session Session) {
	if w == nil {
		return
	}
	line := strings.Repeat("=", 72)
	w.WriteLine(line)
	w.WriteLine("MaaGO core session")
	w.WriteLine("started: " + time.Now().Format("2006-01-02 15:04:05"))
	if session.Version != "" {
		w.WriteLine("core: " + session.Version)
	}
	if session.Address != "" {
		w.WriteLine("address: " + session.Address)
	}
	if session.ADBPath != "" {
		w.WriteLine("adb: " + session.ADBPath)
	}
	if len(session.TaskNames) > 0 {
		w.WriteLine("tasks: " + strings.Join(session.TaskNames, " -> "))
	}
	w.WriteLine(line)
}

// Meta appends one key/value line inside the current session.
func (w *Writer) Meta(key, value string) {
	if w == nil || value == "" {
		return
	}
	w.WriteLine(key + ": " + value)
}

// EndSession writes the closing separator.
func (w *Writer) EndSession(status, message string, duration time.Duration) {
	if w == nil {
		return
	}
	w.WriteLine(fmt.Sprintf("session ended: %s · %s · %s", status, message, duration.Round(time.Second)))
	w.WriteLine(strings.Repeat("=", 72))
	w.WriteLine("")
}

// Close flushes and closes the current file.
func (w *Writer) Close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		_ = w.file.Sync()
		_ = w.file.Close()
		w.file = nil
	}
}

// CurrentPath returns the active log file path for diagnostics.
func (w *Writer) CurrentPath() string {
	if w == nil {
		return ""
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.date == "" {
		return ""
	}
	return filepath.Join(w.dir, "core-"+w.date+".log")
}

func (w *Writer) openLocked(date string) {
	if w.file != nil {
		_ = w.file.Sync()
		_ = w.file.Close()
	}
	path := filepath.Join(w.dir, "core-"+date+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		w.file = nil
		return
	}
	w.file = file
	w.date = date
}

func (w *Writer) prune(days int) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "core-") || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(w.dir, entry.Name()))
		}
	}
}
