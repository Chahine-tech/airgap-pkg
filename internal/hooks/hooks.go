package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"
)

// Run executes a shell hook if the template string is non-empty.
// tmplStr is the command with Go template placeholders (e.g. "trivy image --input {{ .Path }}").
// vars is a map of variables available in the template.
// Returns nil if the hook is empty. Returns an error if the template is invalid or the command fails.
func Run(tmplStr string, vars map[string]string) error {
	if tmplStr == "" {
		return nil
	}
	tmpl, err := template.New("hook").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("hook template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return fmt.Errorf("hook template execute: %w", err)
	}
	cmd := exec.Command("sh", "-c", buf.String())
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
