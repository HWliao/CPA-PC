package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesServesInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	RegisterRoutes(engine, Info{
		Version: "test-version",
		CPA:     CPAInfo{Host: "", Port: 8317},
		Usage:   UsageInfo{Enabled: true},
	})

	req := httptest.NewRequest(http.MethodGet, "/cpa-pc/info", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got Info
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Service != "cpa-pc" {
		t.Fatalf("Service = %q, want %q", got.Service, "cpa-pc")
	}
	if got.Version != "test-version" {
		t.Fatalf("Version = %q, want %q", got.Version, "test-version")
	}
	if got.CPA.Port != 8317 {
		t.Fatalf("CPA.Port = %d, want 8317", got.CPA.Port)
	}
	if !got.Usage.Enabled {
		t.Fatal("Usage.Enabled = false, want true")
	}
}
