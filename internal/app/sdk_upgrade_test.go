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
management: {secret-key: "test-management-key"}
oauth: {auth-dir: "./auth"}
requests: {tls-handshake-timeout-seconds: 12}
routing: {session-affinity-rebind-on: "quota"}
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
			sdkCfg.TLSHandshakeTimeoutSeconds != 12 || sdkCfg.Routing.SessionAffinityRebindOn != "quota" ||
			!sdkCfg.Client.Claude.ShowCodexCommentary {
			t.Fatal("CPA-PC did not preserve the configured SDK settings")
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
