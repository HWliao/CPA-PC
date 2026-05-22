package app

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	cpaVersionHeader   = "X-CPA-VERSION"
	cpaBuildDateHeader = "X-CPA-BUILD-DATE"
)

func buildInfoHeaderMiddleware(buildDate string) gin.HandlerFunc {
	sdkVersion := resolveCLIProxyAPIVersion()
	buildDate = strings.TrimSpace(buildDate)

	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path != "/v0/management" && !strings.HasPrefix(path, "/v0/management/") {
			c.Next()
			return
		}

		c.Writer = &buildInfoResponseWriter{
			ResponseWriter: c.Writer,
			sdkVersion:     sdkVersion,
			buildDate:      buildDate,
		}
		c.Next()
	}
}

type buildInfoResponseWriter struct {
	gin.ResponseWriter
	sdkVersion string
	buildDate  string
}

func (w *buildInfoResponseWriter) applyBuildInfoHeaders() {
	if w.sdkVersion != "" {
		w.Header().Set(cpaVersionHeader, w.sdkVersion)
	}
	if w.buildDate != "" && !strings.EqualFold(w.buildDate, "unknown") {
		w.Header().Set(cpaBuildDateHeader, w.buildDate)
		return
	}
	w.Header().Del(cpaBuildDateHeader)
}

func (w *buildInfoResponseWriter) WriteHeader(code int) {
	w.applyBuildInfoHeaders()
	w.ResponseWriter.WriteHeader(code)
}

func (w *buildInfoResponseWriter) WriteHeaderNow() {
	w.applyBuildInfoHeaders()
	w.ResponseWriter.WriteHeaderNow()
}

func (w *buildInfoResponseWriter) Write(data []byte) (int, error) {
	w.applyBuildInfoHeaders()
	return w.ResponseWriter.Write(data)
}

func (w *buildInfoResponseWriter) WriteString(data string) (int, error) {
	w.applyBuildInfoHeaders()
	return w.ResponseWriter.WriteString(data)
}
