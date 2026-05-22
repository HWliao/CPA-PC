package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildInfoHeaderMiddlewareOverridesManagementHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(buildInfoHeaderMiddleware("2026-05-22T00:00:00Z"))
	router.GET("/v0/management/config", func(c *gin.Context) {
		c.Header(cpaVersionHeader, "dev")
		c.Header(cpaBuildDateHeader, "unknown")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v0/management/config", nil)
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get(cpaVersionHeader); got != resolveCLIProxyAPIVersion() {
		t.Fatalf("%s = %q, want %q", cpaVersionHeader, got, resolveCLIProxyAPIVersion())
	}
	if got := rec.Header().Get(cpaBuildDateHeader); got != "2026-05-22T00:00:00Z" {
		t.Fatalf("%s = %q, want %q", cpaBuildDateHeader, got, "2026-05-22T00:00:00Z")
	}
}

func TestBuildInfoHeaderMiddlewareDropsUnknownBuildDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(buildInfoHeaderMiddleware("unknown"))
	router.GET("/v0/management/config", func(c *gin.Context) {
		c.Header(cpaBuildDateHeader, "unknown")
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v0/management/config", nil)
	router.ServeHTTP(rec, req)

	if got := rec.Header().Get(cpaBuildDateHeader); got != "" {
		t.Fatalf("%s = %q, want empty", cpaBuildDateHeader, got)
	}
}
