package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
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
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.RemoteManagement.SecretKey), []byte(DefaultManagementKey)); err != nil {
		t.Fatalf("example management key does not match default: %v", err)
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

func TestLoadV8ConfigWithCompanion(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`config-version: 8
server:
  host: "127.0.0.1"
  port: 18431
management:
  secret-key: "custom-management-key"
  allow-remote: true
  disable-control-panel: true
oauth:
  auth-dir: "./credentials"
observability:
  logs:
    logging-to-file: true
access:
  api-keys: ["client-key"]
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cpa-pc.yaml"), []byte(`data-dir: "./state"
logs-dir: "./logs-custom"
static-dir: "./assets"
usage:
  enabled: false
  db-path: "./state/stats.sqlite"
  query-limit: 42
`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 18431 || !cfg.RemoteManagement.AllowRemote || !cfg.RemoteManagement.DisableControlPanel || !cfg.LoggingToFile {
		t.Fatalf("SDK settings not loaded from v8 config: %+v", cfg)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(cfg.RemoteManagement.SecretKey), []byte("custom-management-key")); err != nil {
		t.Fatalf("management key does not match: %v", err)
	}
	if cfg.Runtime.AuthDir != filepath.Join(dir, "credentials") || cfg.Runtime.DataDir != filepath.Join(dir, "state") || cfg.Runtime.LogsDir != filepath.Join(dir, "logs-custom") || cfg.Runtime.StaticDir != filepath.Join(dir, "assets") || cfg.Runtime.UsageDBPath != filepath.Join(dir, "state", "stats.sqlite") {
		t.Fatalf("runtime paths not preserved: %+v", cfg.Runtime)
	}
	if cfg.Usage.Enabled || cfg.Usage.QueryLimit != 42 {
		t.Fatalf("usage settings not loaded from companion: %+v", cfg.Usage)
	}
	data, err := os.ReadFile(configPath)
	if err != nil || string(data) != string(content) {
		t.Fatalf("reading config changed file: %v", err)
	}
}

func TestLoadV8ConfigRequiresCompanion(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte("config-version: 8\nserver: {port: 18431}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(configPath)
	if err == nil || !strings.Contains(err.Error(), "cpa-pc.yaml") {
		t.Fatalf("Load() error = %v, want missing companion error", err)
	}
}

func TestLoadV8ConfigRespectsExplicitEmptyManagementKey(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("config-version: 8\nmanagement: {secret-key: \"\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, CPAConfigFileName), []byte("usage: {enabled: false}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RemoteManagement.SecretKey != "" {
		t.Fatal("explicit empty v8 management key was replaced with default")
	}
}

func TestLoadLegacyConfigWithCompanionOverridesOnlyCPASettings(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 18431\ndata-dir: ./old-data\nusage:\n  enabled: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cpa-pc.yaml"), []byte("data-dir: ./new-data\nusage:\n  enabled: false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 18431 || cfg.Runtime.DataDir != filepath.Join(dir, "new-data") || cfg.Usage.Enabled {
		t.Fatalf("Load() = %+v, want SDK port and companion CPA settings", cfg)
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
