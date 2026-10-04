package app

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	"github.com/gin-gonic/gin"
)

func protectUnmigratedConfig(configPath string) gin.HandlerFunc {
	companionPath := filepath.Join(filepath.Dir(configPath), pcconfig.CPAConfigFileName)
	return func(c *gin.Context) {
		method, path := c.Request.Method, c.Request.URL.Path
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}
		isV8Write := path == "/v8/management/config" || path == "/v8/management/config.yaml" ||
			strings.HasPrefix(path, "/v8/management/config/") || strings.HasPrefix(path, "/v8/management/plugins/") ||
			path == "/v8/management/plugins"
		isLegacyYAMLWrite := method == http.MethodPut && path == "/v0/management/config.yaml"
		if !isV8Write && !isLegacyYAMLWrite {
			c.Next()
			return
		}
		if _, err := os.Stat(companionPath); err == nil {
			c.Next()
			return
		} else if !os.IsNotExist(err) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "read_cpa_pc_config_failed"})
			return
		}
		if isLegacyYAMLWrite {
			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid_config"})
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			v8, err := pcconfig.IsV8Config(body)
			if err != nil || !v8 {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "migrate_cpa_pc_config_first", "message": "migrate CPA-PC settings to cpa-pc.yaml before writing v8 config"})
	}
}
