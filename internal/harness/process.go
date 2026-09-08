package harness

import (
	"os/exec"
	"runtime"
	"strings"
)

// IsCursorRunning reports whether the Cursor IDE main process is running.
func IsCursorRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq Cursor.exe", "/NH").Output()
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "cursor.exe") {
				return true
			}
		}
		return false
	case "darwin":
		err := exec.Command("pgrep", "-f", "Cursor.app/Contents/MacOS/Cursor").Run()
		return err == nil
	default:
		out, err := exec.Command("pgrep", "-f", "cursor").Output()
		if err != nil {
			return false
		}
		for _, pid := range strings.Fields(string(out)) {
			cmdline, err := exec.Command("ps", "-p", pid, "-o", "args=").Output()
			if err != nil {
				continue
			}
			if strings.Contains(string(cmdline), "--type=") {
				continue
			}
			return true
		}
		return false
	}
}

// IsChatGPTRunning reports whether the ChatGPT desktop app main process is running.
func IsChatGPTRunning() bool {
	switch runtime.GOOS {
	case "windows":
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq ChatGPT.exe", "/NH").Output()
		if err != nil {
			return false
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "chatgpt.exe") {
				return true
			}
		}
		return false
	case "darwin":
		err := exec.Command("pgrep", "-f", "ChatGPT.app/Contents/MacOS/ChatGPT").Run()
		return err == nil
	default:
		out, err := exec.Command("pgrep", "-f", "chatgpt").Output()
		if err != nil {
			out, err = exec.Command("pgrep", "-f", "ChatGPT").Output()
			if err != nil {
				return false
			}
		}
		for _, pid := range strings.Fields(string(out)) {
			cmdline, err := exec.Command("ps", "-p", pid, "-o", "args=").Output()
			if err != nil {
				continue
			}
			if strings.Contains(string(cmdline), "--type=") {
				continue
			}
			return true
		}
		return false
	}
}
