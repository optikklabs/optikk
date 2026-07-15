// Package browser opens URLs in the user's default browser.
package browser

import (
	"os/exec"
	"runtime"
)

// Open best-effort opens url in the default browser. Failure is deliberately
// non-fatal and silent: callers print the URL too, so a headless machine or a
// missing opener degrades to "click this link" rather than an error.
func Open(url string) {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	_ = exec.Command(name, args...).Start()
}
