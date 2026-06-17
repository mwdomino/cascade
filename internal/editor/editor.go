package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func EditorCmd() string {
	if e := strings.TrimSpace(os.Getenv("EDITOR")); e != "" {
		return e
	}
	if e := strings.TrimSpace(os.Getenv("VISUAL")); e != "" {
		return e
	}
	return "vi"
}

func Open(path string) error {
	editorLine := EditorCmd()
	parts := strings.Fields(editorLine)
	if len(parts) == 0 {
		return fmt.Errorf("no editor configured")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
