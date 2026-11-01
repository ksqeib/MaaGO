//go:build !windows

package notify

import "errors"

// Send is unavailable outside Windows.
func Send(title, message string) error {
	return errors.New("system notifications are only supported on Windows")
}
