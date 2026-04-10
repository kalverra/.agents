package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Detect checks whether the given agent is installed on this machine.
func Detect(a Agent, verbose bool) bool {
	switch a {
	case Claude:
		return detectClaude(verbose)
	case Gemini:
		return detectGemini(verbose)
	case Antigravity:
		return detectAntigravity(verbose)
	case Cursor:
		return detectCursor(verbose)
	default:
		return false
	}
}

// DetectAll returns a map of which agents are detected.
func DetectAll(verbose bool) map[Agent]bool {
	m := make(map[Agent]bool, len(All()))
	for _, a := range All() {
		m[a] = Detect(a, verbose)
	}
	return m
}

func vlog(verbose bool, format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}

func detectClaude(verbose bool) bool {
	if p, err := exec.LookPath("claude"); err == nil {
		vlog(verbose, "claude: found in PATH (%s)", p)
		return true
	}
	d := filepath.Join(homeDir(), ".claude")
	if isDir(d) {
		vlog(verbose, "claude: ~/.claude exists")
		return true
	}
	return false
}

func detectGemini(verbose bool) bool {
	for _, name := range []string{"gemini", "gemini-cli"} {
		if p, err := exec.LookPath(name); err == nil {
			vlog(verbose, "%s: found in PATH (%s)", name, p)
			return true
		}
	}
	gd := GeminiConfigDir()
	if isDir(gd) {
		vlog(verbose, "gemini: config dir exists (%s)", gd)
		return true
	}
	return false
}

func detectAntigravity(verbose bool) bool {
	if p, err := exec.LookPath("antigravity"); err == nil {
		vlog(verbose, "antigravity: found in PATH (%s)", p)
		return true
	}
	if isDir(filepath.Join(homeDir(), ".antigravity-server")) {
		vlog(verbose, "antigravity: ~/.antigravity-server exists")
		return true
	}
	switch runtime.GOOS {
	case "darwin":
		ag := filepath.Join(homeDir(), "Library", "Application Support", "Antigravity")
		if isDir(ag) {
			vlog(verbose, "antigravity: %s exists", ag)
			return true
		}
	case "linux":
		ag := filepath.Join(homeDir(), ".config", "Antigravity")
		if isDir(ag) {
			vlog(verbose, "antigravity: %s exists", ag)
			return true
		}
	}
	return false
}

func detectCursor(verbose bool) bool {
	if isDir(filepath.Join(homeDir(), ".cursor")) {
		vlog(verbose, "cursor: ~/.cursor exists")
		return true
	}
	switch runtime.GOOS {
	case "darwin":
		p := filepath.Join(homeDir(), "Library", "Application Support", "Cursor")
		if isDir(p) {
			vlog(verbose, "cursor: %s exists", p)
			return true
		}
	case "linux":
		p := filepath.Join(homeDir(), ".config", "Cursor")
		if isDir(p) {
			vlog(verbose, "cursor: %s exists", p)
			return true
		}
	}
	return false
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}
