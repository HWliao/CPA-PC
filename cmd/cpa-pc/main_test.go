package main

import (
	"bytes"
	"context"
	"os"
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

func TestRunPreviewThenMigrateConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("port: 18431\nremote-management:\n  secret-key: test-secret\nusage:\n  enabled: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run(context.Background(), []string{"-config", path, "-preview-config-migration"}, &stdout, &stderr); code != 0 {
		t.Fatalf("preview returned %d: %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "test-secret") || !strings.Contains(stdout.String(), "cpa-pc.yaml") {
		t.Fatalf("preview leaked a key or omitted companion path: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "cpa-pc.yaml")); !os.IsNotExist(err) {
		t.Fatalf("preview created companion: %v", err)
	}
	stdout.Reset()
	if code := run(context.Background(), []string{"-config", path, "-migrate-config"}, &stdout, &stderr); code != 0 {
		t.Fatalf("migration returned %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "config migrated to v8") {
		t.Fatalf("migration output = %q", stdout.String())
	}
}
