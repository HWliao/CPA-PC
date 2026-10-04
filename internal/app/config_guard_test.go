package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProtectUnmigratedConfigRequiresCompanionForV8Writes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	router := gin.New()
	router.Use(protectUnmigratedConfig(filepath.Join(dir, "config.yaml")))
	router.POST("/v8/management/config", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/v8/management/oauth/callback", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/v8/management/plugins/store/example/install", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/v8/management/config", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/v8/management/cpa-pc/usage/import", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodPost, "/v8/management/config", http.StatusConflict},
		{http.MethodPost, "/v8/management/plugins/store/example/install", http.StatusConflict},
		{http.MethodPost, "/v8/management/oauth/callback", http.StatusOK},
		{http.MethodGet, "/v8/management/config", http.StatusOK},
		{http.MethodPost, "/v8/management/cpa-pc/usage/import", http.StatusOK},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.want {
			t.Fatalf("%s %s returned %d, want %d", tc.method, tc.path, rec.Code, tc.want)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "cpa-pc.yaml"), []byte("usage: {enabled: true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v8/management/config", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("v8 write with companion returned %d, want 200", rec.Code)
	}
}

func TestProtectUnmigratedConfigRejectsV8YAMLOnLegacyRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(protectUnmigratedConfig(filepath.Join(t.TempDir(), "config.yaml")))
	router.PUT("/v0/management/config.yaml", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(body))
	})

	for _, tc := range []struct {
		body string
		want int
	}{
		{"port: 8317\n", http.StatusOK},
		{"config-version: 8\nserver: {port: 8317}\n", http.StatusConflict},
	} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/v0/management/config.yaml", strings.NewReader(tc.body)))
		if rec.Code != tc.want {
			t.Fatalf("PUT config.yaml returned %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
		}
		if tc.want == http.StatusOK && rec.Body.String() != tc.body {
			t.Fatalf("legacy YAML body changed to %q", rec.Body.String())
		}
	}
}
