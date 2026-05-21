package app

import (
	"os"
	"path/filepath"
	"testing"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
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
