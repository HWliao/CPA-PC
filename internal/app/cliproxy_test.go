package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

func TestLoadCPAConfigHashesDefaultManagementKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`host: "127.0.0.1"
port: 18317
auth-dir: "./auths"
remote-management:
  secret-key: ""
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := pcconfig.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	cpaCfg, err := loadCPAConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if cpaCfg.RemoteManagement.SecretKey == pcconfig.DefaultManagementKey {
		t.Fatal("CPA management key was left in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cpaCfg.RemoteManagement.SecretKey), []byte(pcconfig.DefaultManagementKey)); err != nil {
		t.Fatalf("default management key does not verify against CPA hash: %v", err)
	}
	if cpaCfg.AuthDir != filepath.Join(dir, "auths") {
		t.Fatalf("AuthDir = %q, want %q", cpaCfg.AuthDir, filepath.Join(dir, "auths"))
	}
}

func TestConfigureManagementStaticPathSetsDefaultWhenUnset(t *testing.T) {
	t.Setenv("MANAGEMENT_STATIC_PATH", "")
	staticDir := filepath.Join(t.TempDir(), "static")

	if err := configureManagementStaticPath(staticDir); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("MANAGEMENT_STATIC_PATH"); got != staticDir {
		t.Fatalf("MANAGEMENT_STATIC_PATH = %q, want %q", got, staticDir)
	}
}

func TestConfigureManagementStaticPathPreservesExplicitEnv(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "env-static")
	t.Setenv("MANAGEMENT_STATIC_PATH", existing)

	if err := configureManagementStaticPath(filepath.Join(t.TempDir(), "config-static")); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("MANAGEMENT_STATIC_PATH"); got != existing {
		t.Fatalf("MANAGEMENT_STATIC_PATH = %q, want %q", got, existing)
	}
}

func TestConfigureEmbeddedLogOutputWritesMainLog(t *testing.T) {
	t.Setenv("WRITABLE_PATH", "")
	t.Setenv("writable_path", "")

	logger := log.StandardLogger()
	originalOut := logger.Out
	originalFormatter := logger.Formatter
	originalReportCaller := logger.ReportCaller
	t.Cleanup(func() {
		log.SetOutput(originalOut)
		log.SetFormatter(originalFormatter)
		log.SetReportCaller(originalReportCaller)
	})

	dir := t.TempDir()
	cfg := &pcconfig.Config{
		LoggingToFile: true,
		Runtime: pcconfig.RuntimePaths{
			BaseDir: dir,
		},
	}

	if err := ensureEmbeddedWritablePath(cfg); err != nil {
		t.Fatal(err)
	}
	cleanup, err := configureEmbeddedLogOutput(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cleanup == nil {
		t.Fatal("cleanup = nil, want configured log writer cleanup")
	}

	log.WithField("request_id", "abcdef12").Info("embedded log check")
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(dir, "logs", "main.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file %s: %v", logPath, err)
	}
	if !bytes.Contains(data, []byte("embedded log check")) {
		t.Fatalf("main.log = %q, want embedded log message", string(data))
	}
	if !strings.Contains(string(data), "[abcdef12]") {
		t.Fatalf("main.log = %q, want request id", string(data))
	}
}

func TestEnsureEmbeddedWritablePathPreservesExplicitEnv(t *testing.T) {
	existing := filepath.Join(t.TempDir(), "writable")
	t.Setenv("WRITABLE_PATH", existing)

	if err := ensureEmbeddedWritablePath(&pcconfig.Config{Runtime: pcconfig.RuntimePaths{BaseDir: t.TempDir()}}); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("WRITABLE_PATH"); got != existing {
		t.Fatalf("WRITABLE_PATH = %q, want %q", got, existing)
	}
}
