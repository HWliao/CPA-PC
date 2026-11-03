package app

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

func TestMigrateConfigPreservesSettingsAndBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	previous := []byte(`host: "127.0.0.1"
port: 18431
auth-dir: "./credentials"
remote-management:
  secret-key: "custom-key"
  allow-remote: true
data-dir: "./state"
logs-dir: "./custom-logs"
static-dir: "./assets"
usage:
  enabled: false
  db-path: "./state/stats.sqlite"
  query-limit: 42
logging-to-file: true
request-retry: 0
`)
	if err := os.WriteFile(path, previous, 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewConfigMigration(path)
	if err != nil {
		t.Fatal(err)
	}
	if preview.AlreadyV8 || preview.CompanionPath != filepath.Join(dir, "cpa-pc.yaml") {
		t.Fatalf("preview = %+v", preview)
	}
	if _, err := os.Stat(preview.CompanionPath); !os.IsNotExist(err) {
		t.Fatalf("preview created companion: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != string(previous) {
		t.Fatalf("preview changed original: %v", err)
	}

	result, err := MigrateConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, preview) {
		t.Fatalf("migration result = %+v, want %+v", result, preview)
	}
	backup, err := os.ReadFile(preview.BackupPath)
	if err != nil || string(backup) != string(previous) {
		t.Fatalf("backup not preserved: %v", err)
	}
	v8Data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := yaml.Unmarshal(v8Data, &root); err != nil {
		t.Fatal(err)
	}
	if root["config-version"] != 8 || root["data-dir"] != nil || root["usage"] != nil {
		t.Fatalf("v8 file still contains CPA-PC settings: %s", v8Data)
	}
	sdkCfg, err := sdkconfig.ParseConfigBytes(v8Data)
	if err != nil {
		t.Fatal(err)
	}
	if sdkCfg.Port != 18431 || sdkCfg.Host != "127.0.0.1" || sdkCfg.RequestRetry != 0 {
		t.Fatalf("SDK config changed: %+v", sdkCfg)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(sdkCfg.RemoteManagement.SecretKey), []byte("custom-key")); err != nil {
		t.Fatalf("management key changed: %v", err)
	}
	cfg, err := pcconfig.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Usage.Enabled || cfg.Usage.QueryLimit != 42 || cfg.Runtime.UsageDBPath != filepath.Join(dir, "state", "stats.sqlite") || cfg.Runtime.AuthDir != filepath.Join(dir, "credentials") || !cfg.LoggingToFile {
		t.Fatalf("migrated CPA-PC config changed: %+v", cfg)
	}
	repeated, err := MigrateConfig(path)
	if err != nil || !repeated.AlreadyV8 {
		t.Fatalf("repeat migration = %+v, %v", repeated, err)
	}
	if err := os.WriteFile(path, backup, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(preview.CompanionPath); err != nil {
		t.Fatal(err)
	}
	rolledBack, err := pcconfig.Load(path)
	if err != nil || rolledBack.Port != 18431 || rolledBack.Usage.Enabled || rolledBack.Runtime.UsageDBPath != filepath.Join(dir, "state", "stats.sqlite") {
		t.Fatalf("rollback did not restore legacy settings: %+v, %v", rolledBack, err)
	}
}

func TestMigrateConfigRequiresUnusedCompanionAndBackup(t *testing.T) {
	for _, name := range []string{"cpa-pc.yaml", "config.yaml.v0.bak"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte("port: 18431\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), []byte("existing"), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := MigrateConfig(path); err == nil || !strings.Contains(err.Error(), "already exists") {
				t.Fatalf("migration error = %v, want existing target", err)
			}
		})
	}
}

func TestMigrateConfigPreservesDefaultAndHashedManagementKeys(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("already-hashed"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, source, key string
	}{
		{"default", "port: 18431\n", pcconfig.DefaultManagementKey},
		{"hashed", "remote-management:\n  secret-key: " + string(hash) + "\n", "already-hashed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(tc.source), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := MigrateConfig(path); err != nil {
				t.Fatal(err)
			}
			cfg, err := pcconfig.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := bcrypt.CompareHashAndPassword([]byte(cfg.RemoteManagement.SecretKey), []byte(tc.key)); err != nil {
				t.Fatalf("management key changed: %v", err)
			}
		})
	}
}

func TestMigrateConfigReportsUnmappedSDKSectionsBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("port: 18431\nunknown-sdk-setting: keep-me\nusage: {enabled: false}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewConfigMigration(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview.UnknownSections, []string{"unknown-sdk-setting"}) {
		t.Fatalf("unmapped sections = %v", preview.UnknownSections)
	}
	if _, err := MigrateConfig(path); err == nil || !strings.Contains(err.Error(), "unknown-sdk-setting") {
		t.Fatalf("migration error = %v, want unmapped section", err)
	}
	if _, err := os.Stat(preview.BackupPath); !os.IsNotExist(err) {
		t.Fatalf("blocked migration created backup: %v", err)
	}
}

func TestMigrateConfigDoesNotTreatKnownLegacySectionsAsUnmapped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`# streaming:
#   keepalive-seconds: 0
streaming:
  keepalive-seconds: 15
# codex-header-defaults:
codex-header-defaults:
  user-agent: "codex-test"
claude-api-key: []
codex-api-key: []
openai-compatibility: []
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewConfigMigration(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.UnknownSections) != 0 {
		t.Fatalf("known SDK sections flagged as unknown: %v", preview.UnknownSections)
	}
	if preview.SDKSettingsChanged {
		t.Fatal("migration changed effective SDK settings")
	}
}

func TestExampleConfigUsesV8WithCompanion(t *testing.T) {
	example, err := os.ReadFile(filepath.Join("..", "..", "config.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	v8, err := pcconfig.IsV8Config(example)
	if err != nil || !v8 {
		t.Fatalf("example is not v8: %v", err)
	}
	if _, err := sdkconfig.ParseConfigBytes(example); err != nil {
		t.Fatalf("SDK cannot parse example: %v", err)
	}
	cfg, err := pcconfig.Load(filepath.Join("..", "..", "config.example.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Usage.Enabled || cfg.Runtime.UsageDBPath == "" {
		t.Fatalf("example companion settings not loaded: %+v", cfg.Usage)
	}
}
