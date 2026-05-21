package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`host: ""
port: 0
remote-management:
  secret-key: ""
usage:
  db-path: "./custom/usage.sqlite"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Host != DefaultHost {
		t.Fatalf("Host = %q, want %q", cfg.Host, DefaultHost)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.RemoteManagement.SecretKey != DefaultManagementKey {
		t.Fatalf("RemoteManagement.SecretKey = %q, want default", cfg.RemoteManagement.SecretKey)
	}
	if cfg.RemoteManagement.DisableControlPanel {
		t.Fatal("RemoteManagement.DisableControlPanel = true, want false")
	}
	if !cfg.Usage.Enabled {
		t.Fatal("Usage.Enabled = false, want true")
	}
	if cfg.Usage.QueryLimit != DefaultUsageQueryLimit {
		t.Fatalf("Usage.QueryLimit = %d, want %d", cfg.Usage.QueryLimit, DefaultUsageQueryLimit)
	}

	wantDBPath := filepath.Join(dir, "custom", "usage.sqlite")
	if cfg.Runtime.UsageDBPath != wantDBPath {
		t.Fatalf("Runtime.UsageDBPath = %q, want %q", cfg.Runtime.UsageDBPath, wantDBPath)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	wantAuthDir := filepath.Join(homeDir, ".cli-proxy-api")
	if cfg.Runtime.AuthDir != wantAuthDir {
		t.Fatalf("Runtime.AuthDir = %q, want %q", cfg.Runtime.AuthDir, wantAuthDir)
	}
}

func TestLoadAllowsExplicitFalseDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`remote-management:
  disable-control-panel: false
usage:
  enabled: false
logging-to-file: false
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.RemoteManagement.DisableControlPanel {
		t.Fatal("RemoteManagement.DisableControlPanel = true, want false")
	}
	if cfg.Usage.Enabled {
		t.Fatal("Usage.Enabled = true, want false")
	}
	if cfg.LoggingToFile {
		t.Fatal("LoggingToFile = true, want false")
	}
}

func TestLoadExampleConfig(t *testing.T) {
	configPath := filepath.Join("..", "..", "config.example.yaml")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Host != DefaultHost {
		t.Fatalf("Host = %q, want %q", cfg.Host, DefaultHost)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("Port = %d, want %d", cfg.Port, DefaultPort)
	}
	if cfg.RemoteManagement.SecretKey != DefaultManagementKey {
		t.Fatalf("RemoteManagement.SecretKey = %q, want %q", cfg.RemoteManagement.SecretKey, DefaultManagementKey)
	}
	if cfg.RemoteManagement.DisableControlPanel {
		t.Fatal("RemoteManagement.DisableControlPanel = true, want CPA default false")
	}
	if cfg.LoggingToFile {
		t.Fatal("LoggingToFile = true, want CPA default false")
	}
	if cfg.AuthDir != DefaultAuthDir {
		t.Fatalf("AuthDir = %q, want %q", cfg.AuthDir, DefaultAuthDir)
	}
	if cfg.Usage.DBPath != DefaultUsageDBPath {
		t.Fatalf("Usage.DBPath = %q, want %q", cfg.Usage.DBPath, DefaultUsageDBPath)
	}
}

func TestListenAddressKeepsEmptyCPAHost(t *testing.T) {
	cfg := Config{Host: "", Port: 8317}

	if got := cfg.ListenAddress(); got != ":8317" {
		t.Fatalf("ListenAddress() = %q, want %q", got, ":8317")
	}
}

func TestRuntimeAuthDirExpandsTildeLikeCPA(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("auth-dir: ~/.cli-proxy-api\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(homeDir, ".cli-proxy-api")
	if cfg.Runtime.AuthDir != want {
		t.Fatalf("Runtime.AuthDir = %q, want %q", cfg.Runtime.AuthDir, want)
	}
}

func TestLoadResolvesRuntimePathsFromConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	absStaticDir := filepath.Join(t.TempDir(), "assets")
	content := []byte(`data-dir: "state"
logs-dir: "logz"
static-dir: "` + filepath.ToSlash(absStaticDir) + `"
auth-dir: "auths"
usage:
  db-path: "usage.sqlite"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]string{
		"BaseDir":     dir,
		"ConfigPath":  configPath,
		"DataDir":     filepath.Join(dir, "state"),
		"LogsDir":     filepath.Join(dir, "logz"),
		"StaticDir":   absStaticDir,
		"AuthDir":     filepath.Join(dir, "auths"),
		"UsageDBPath": filepath.Join(dir, "usage.sqlite"),
	}
	actual := map[string]string{
		"BaseDir":     cfg.Runtime.BaseDir,
		"ConfigPath":  cfg.Runtime.ConfigPath,
		"DataDir":     cfg.Runtime.DataDir,
		"LogsDir":     cfg.Runtime.LogsDir,
		"StaticDir":   cfg.Runtime.StaticDir,
		"AuthDir":     cfg.Runtime.AuthDir,
		"UsageDBPath": cfg.Runtime.UsageDBPath,
	}

	for name, want := range expected {
		if actual[name] != filepath.Clean(want) {
			t.Fatalf("%s = %q, want %q", name, actual[name], filepath.Clean(want))
		}
	}
}

func TestLoadMissingConfigReturnsActionableError(t *testing.T) {
	missingConfig := filepath.Join(t.TempDir(), "config.yaml")

	_, err := Load(missingConfig)
	if err == nil {
		t.Fatal("Load returned nil error, want missing config error")
	}

	var missing MissingConfigError
	if !errors.As(err, &missing) {
		t.Fatalf("error = %T, want MissingConfigError", err)
	}
	if missing.ConfigPath != missingConfig {
		t.Fatalf("MissingConfigError.ConfigPath = %q, want %q", missing.ConfigPath, missingConfig)
	}
	if !strings.Contains(err.Error(), "config.example.yaml") {
		t.Fatalf("error = %q, want config.example.yaml guidance", err.Error())
	}
}
