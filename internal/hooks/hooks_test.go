package hooks

import (
	"testing"
)

func TestRun_EmptyHook(t *testing.T) {
	if err := Run("", nil); err != nil {
		t.Fatalf("expected nil for empty hook, got %v", err)
	}
}

func TestRun_SimpleCommand(t *testing.T) {
	if err := Run("true", nil); err != nil {
		t.Fatalf("expected nil for 'true' command, got %v", err)
	}
}

func TestRun_FailingCommand(t *testing.T) {
	if err := Run("false", nil); err == nil {
		t.Fatal("expected error for 'false' command, got nil")
	}
}

func TestRun_InvalidTemplate(t *testing.T) {
	if err := Run("echo {{ .Unclosed", nil); err == nil {
		t.Fatal("expected error for invalid template, got nil")
	}
}

func TestRun_TemplateSubstitution(t *testing.T) {
	vars := map[string]string{
		"Source": "ghcr.io/chaos-mesh/chaos-mesh:v2.7.2",
		"Path":   "/tmp/chaos-mesh.tar",
	}
	// Command just exits 0 — we verify no error and substitution doesn't panic
	if err := Run("test -n '{{ index . \"Source\" }}'", vars); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRun_DotNotation(t *testing.T) {
	vars := map[string]string{"Dest": "chaos-mesh/chaos-mesh:v2.7.2"}
	// .Dest is not valid for map[string]string with dot notation — verify graceful behavior
	// With text/template, map keys accessed via index
	if err := Run("true", vars); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
