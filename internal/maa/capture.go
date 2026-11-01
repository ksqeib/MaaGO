package maa

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/sys/windows"
)

// Capture redirects the process stdout/stderr to a pipe. It must be started
// before MaaCore.dll is loaded so the CRT initialises its std::cout against the
// pipe handle.
type Capture struct {
	read   *os.File
	write  *os.File
	oldOut windows.Handle
	oldErr windows.Handle
	done   chan struct{}
	once   sync.Once
}

// StartStdoutCapture installs the pipe and starts a line reader writing to dst.
func StartStdoutCapture(dst io.Writer) (*Capture, error) {
	read, write, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	capture := &Capture{
		read:  read,
		write: write,
		done:  make(chan struct{}),
	}
	capture.oldOut, _ = windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	capture.oldErr, _ = windows.GetStdHandle(windows.STD_ERROR_HANDLE)

	if err := windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(write.Fd())); err != nil {
		_ = read.Close()
		_ = write.Close()
		return nil, fmt.Errorf("set stdout: %w", err)
	}
	if err := windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(write.Fd())); err != nil {
		_ = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, capture.oldOut)
		_ = read.Close()
		_ = write.Close()
		return nil, fmt.Errorf("set stderr: %w", err)
	}

	go func() {
		defer close(capture.done)
		scanner := bufio.NewScanner(read)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			if _, err := dst.Write([]byte(scanner.Text() + "\n")); err != nil {
				return
			}
		}
	}()

	return capture, nil
}

// Stop restores the original handles and stops the reader.
func (c *Capture) Stop() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		_ = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, c.oldOut)
		_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, c.oldErr)
		_ = c.write.Close()
		_ = c.read.Close()
		<-c.done
	})
}
