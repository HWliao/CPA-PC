package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
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

func TestNewCLIProxyServiceAppliesInitialDebugLogLevel(t *testing.T) {
	t.Setenv("WRITABLE_PATH", "")
	t.Setenv("writable_path", "")
	t.Setenv("MANAGEMENT_STATIC_PATH", "")

	originalLevel := log.GetLevel()
	t.Cleanup(func() {
		log.SetLevel(originalLevel)
	})
	log.SetLevel(log.InfoLevel)

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(`host: "127.0.0.1"
port: 18317
auth-dir: "./auth"
debug: true
logging-to-file: false
usage:
  enabled: false
remote-management:
  secret-key: "123456"
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := pcconfig.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewCLIProxyService(cfg, ServiceOptions{}); err != nil {
		t.Fatal(err)
	}

	if got := log.GetLevel(); got != log.DebugLevel {
		t.Fatalf("log level = %s, want %s", got, log.DebugLevel)
	}
}

func TestSDKStartupRegistersBuiltInTranslators(t *testing.T) {
	raw := []byte(`{"model":"gpt-5.5","messages":[{"role":"user","content":"hello"}]}`)

	out := sdktranslator.TranslateRequest(sdktranslator.FormatOpenAI, sdktranslator.FormatCodex, "gpt-5.5", raw, false)

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("translated payload is not JSON: %v", err)
	}
	if store, ok := payload["store"].(bool); !ok || store {
		t.Fatalf("store = %#v, want false", payload["store"])
	}
	if _, ok := payload["input"]; !ok {
		t.Fatalf("translated payload missing Responses input: %s", out)
	}
	if _, ok := payload["messages"]; ok {
		t.Fatalf("translated payload still contains Chat Completions messages: %s", out)
	}
}

func TestCLIProxyServiceWritesRequestErrorLogWhenRequestLogDisabled(t *testing.T) {
	t.Setenv("WRITABLE_PATH", "")
	t.Setenv("writable_path", "")
	t.Setenv("MANAGEMENT_STATIC_PATH", "")

	logger := log.StandardLogger()
	originalOut := logger.Out
	originalFormatter := logger.Formatter
	originalReportCaller := logger.ReportCaller
	originalLevel := logger.Level
	t.Cleanup(func() {
		log.SetOutput(originalOut)
		log.SetFormatter(originalFormatter)
		log.SetReportCaller(originalReportCaller)
		log.SetLevel(originalLevel)
	})

	dir := t.TempDir()
	port := reserveLocalPort(t)
	configPath := filepath.Join(dir, "config.yaml")
	content := []byte(fmt.Sprintf(`host: "127.0.0.1"
port: %d
auth-dir: "./auth"
api-keys:
  - "test-key"
debug: true
logging-to-file: true
request-log: false
usage:
  enabled: false
remote-management:
  secret-key: "123456"
`, port))
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := pcconfig.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewCLIProxyService(cfg, ServiceOptions{})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case errRun := <-errCh:
			if errRun != nil && !errors.Is(errRun, context.Canceled) {
				t.Fatalf("service.Run returned %v", errRun)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("service did not stop after cancellation")
		}
	})

	client := &http.Client{Timeout: 2 * time.Second}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForHTTPStatus(t, client, http.MethodGet, baseURL+"/healthz", "", http.StatusOK)

	status := doTestRequest(t, client, http.MethodPost, baseURL+"/v1/chat/completions", `{}`)
	if status < http.StatusBadRequest {
		t.Fatalf("POST /v1/chat/completions status = %d, want error status", status)
	}

	waitForRequestErrorLog(t, filepath.Join(dir, "logs"))
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

func reserveLocalPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address = %T, want *net.TCPAddr", listener.Addr())
	}
	return addr.Port
}

func waitForHTTPStatus(t *testing.T, client *http.Client, method string, url string, body string, want int) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	var lastStatus int
	for time.Now().Before(deadline) {
		status, err := doTestRequestMaybe(client, method, url, body)
		if err == nil {
			lastStatus = status
			if status == want {
				return
			}
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}

	if lastErr != nil {
		t.Fatalf("%s %s did not return %d: last error: %v", method, url, want, lastErr)
	}
	t.Fatalf("%s %s status = %d, want %d", method, url, lastStatus, want)
}

func doTestRequest(t *testing.T, client *http.Client, method string, url string, body string) int {
	t.Helper()

	status, err := doTestRequestMaybe(client, method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	return status
}

func doTestRequestMaybe(client *http.Client, method string, url string, body string) (int, error) {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, err
	}
	if reader != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func waitForRequestErrorLog(t *testing.T, logDir string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(logDir)
		if err == nil {
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() || !strings.HasPrefix(name, "error-") || !strings.HasSuffix(name, ".log") {
					continue
				}
				data, errRead := os.ReadFile(filepath.Join(logDir, name))
				if errRead != nil {
					t.Fatal(errRead)
				}
				if bytes.Contains(data, []byte("POST")) && bytes.Contains(data, []byte("/v1/chat/completions")) {
					return
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("no request error log found in %s", logDir)
}
