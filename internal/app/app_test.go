package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLoadsConfigAndPrintsFoundationSummary(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("host: 127.0.0.1\nport: 8317\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err := Run(context.Background(), Options{
		ConfigPath: configPath,
		Version:    "test",
		Stdout:     &stdout,
	})

	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "CPA-PC test foundation initialized") {
		t.Fatalf("stdout = %q, want foundation summary", output)
	}
	if !strings.Contains(output, filepath.Clean(configPath)) {
		t.Fatalf("stdout = %q, want config path", output)
	}
}
