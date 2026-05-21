package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersionDoesNotRequireConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"-version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run returned %d, want 0; stderr=%q", code, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "cpa-pc dev" {
		t.Fatalf("stdout = %q, want %q", got, "cpa-pc dev")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunHelpDoesNotRequireConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"-h"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("run returned %d, want 0", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-config") {
		t.Fatalf("help output = %q, want -config flag", stderr.String())
	}
}

func TestRunMissingConfigExplainsHowToStart(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	missingConfig := filepath.Join(t.TempDir(), "config.yaml")

	code := run(context.Background(), []string{"-config", missingConfig}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("run returned %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "config.example.yaml") {
		t.Fatalf("stderr = %q, want config.example.yaml guidance", stderr.String())
	}
}
