package main

import (
	"time"

	win "golang.org/x/sys/windows"
)

// acquireSingleInstanceMutex takes a named object whose lifetime is the guard.
// Requesting SYNCHRONIZE only is enough to detect an existing instance without
// requiring write access to its mutex.
func acquireSingleInstanceMutex(name string, wait time.Duration) (win.Handle, error) {
	ptr, err := win.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	deadline := time.Now().Add(wait)
	for {
		h, err := win.CreateMutexEx(nil, ptr, 0, win.SYNCHRONIZE)
		if err == nil {
			return h, nil
		}
		if h != 0 {
			_ = win.CloseHandle(h)
		}
		if err != win.ERROR_ALREADY_EXISTS {
			return 0, err
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return 0, err
		}
		time.Sleep(min(100*time.Millisecond, remaining))
	}
}
