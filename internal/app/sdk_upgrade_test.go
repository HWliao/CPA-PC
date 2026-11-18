package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	"github.com/gin-gonic/gin"
	sdkapi "github.com/router-for-me/CLIProxyAPI/v8/sdk/api"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
)

func TestSDKConfigWritePreservesNewSettingsAndCompanion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`config-version: 8
server:
  host: "127.0.0.1"
  port: 18317
  max-request-body-bytes: 4096
  response-compression: {enabled: false}
  shutdown-timeout: 9
  kv-store: {disabled: true, path: "custom-kv.db"}
management: {secret-key: "test-management-key"}
oauth: {auth-dir: "./auth"}
requests:
  tls-handshake-timeout-seconds: 12
  payload:
    replace:
      - models: [{name: "*", protocol: "codex"}]
        target: {paths: ["metadata.description"]}
        replacements:
          - pattern: '(?i)secret'
            replacement: ""
routing: {session-affinity: true, session-affinity-rebind-on: "never"}
upstream:
  claude: {server-side-fallback: true}
client:
  claude: {show-codex-commentary: true}
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	companionPath := filepath.Join(dir, pcconfig.CPAConfigFileName)
	companion := []byte("data-dir: ./state\nusage: {enabled: false, db-path: ./state/usage.sqlite}\n")
	if err := os.WriteFile(companionPath, companion, 0o600); err != nil {
		t.Fatal(err)
	}
	load := func(t *testing.T) *sdkconfig.Config {
		t.Helper()
		cfg, err := pcconfig.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		sdkCfg, err := loadCPAConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if sdkCfg.MaxRequestBodyBytes != 4096 || sdkCfg.EffectiveResponseCompression().IsEnabled() ||
			sdkCfg.TLSHandshakeTimeoutSeconds != 12 || sdkCfg.Routing.SessionAffinityRebindOn != "never" ||
			!sdkCfg.Client.Claude.ShowCodexCommentary {
			t.Fatal("CPA-PC did not preserve the configured SDK settings")
		}
		if sdkCfg.ShutdownTimeoutSeconds != 9 || !sdkCfg.KVStore.Disabled || sdkCfg.KVStore.Path != "custom-kv.db" ||
			!sdkCfg.Routing.SessionAffinity || !sdkCfg.Claude.ServerSideFallback {
			t.Fatal("CPA-PC did not preserve the SDK v8.0.77 settings")
		}
		if len(sdkCfg.Payload.Replace) != 1 || len(sdkCfg.Payload.Replace[0].Target.Paths) != 1 ||
			sdkCfg.Payload.Replace[0].Target.Paths[0] != "metadata.description" || len(sdkCfg.Payload.Replace[0].Replacements) != 1 {
			t.Fatal("CPA-PC did not preserve the payload replacement rule")
		}
		replacement := sdkCfg.Payload.Replace[0].Replacements[0]
		if value, ok := replacement["replacement"]; !ok || value != "" || replacement["pattern"] != "(?i)secret" {
			t.Fatal("CPA-PC did not preserve the empty regex replacement")
		}
		if cfg.Usage.Enabled || cfg.Runtime.UsageDBPath != filepath.Join(dir, "state", "usage.sqlite") {
			t.Fatal("CPA-PC companion settings changed")
		}
		return sdkCfg
	}
	sdkCfg := load(t)
	engine := gin.New()
	engine.Use(protectUnmigratedConfig(path))
	handler := sdkapi.NewHandler(sdkCfg, path, nil)
	engine.GET("/v8/management/config.yaml", handler.ConfigV8)
	engine.PUT("/v8/management/config/*path", handler.ConfigV8)
	get := httptest.NewRecorder()
	engine.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/v8/management/config.yaml", nil))
	etag := get.Header().Get("ETag")
	if get.Code != http.StatusOK || etag == "" {
		t.Fatalf("GET config: status=%d ETag=%q", get.Code, etag)
	}
	for _, tc := range []struct {
		name    string
		ifMatch string
		status  int
		port    int
	}{
		{name: "stale", ifMatch: `"stale"`, status: http.StatusPreconditionFailed, port: 18317},
		{name: "matching", ifMatch: etag, status: http.StatusOK, port: 18318},
		{name: "unconditional", status: http.StatusOK, port: 18318},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPut, "/v8/management/config/server/port", strings.NewReader("18318"))
			req.Header.Set("Content-Type", "application/json")
			if tc.ifMatch != "" {
				req.Header.Set("If-Match", tc.ifMatch)
			}
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			if response.Code != tc.status {
				t.Fatalf("PUT config: status=%d body=%s", response.Code, response.Body.String())
			}
			if cfg := load(t); cfg.Port != tc.port {
				t.Fatalf("port = %d, want %d", cfg.Port, tc.port)
			}
		})
	}
	got, err := os.ReadFile(companionPath)
	if err != nil || !bytes.Equal(got, companion) {
		t.Fatalf("SDK config write changed the CPA-PC companion: %v", err)
	}
}
