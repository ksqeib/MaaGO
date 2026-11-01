//go:build windows

package notify

import toast "git.sr.ht/~jackmordaunt/go-toast/v2"

// Send shows a Windows toast notification.
func Send(title, message string) error {
	notification := toast.Notification{
		AppID: "MaaGO",
		Title: title,
		Body:  message,
	}
	return notification.Push()
}
