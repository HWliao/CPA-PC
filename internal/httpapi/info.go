package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Info struct {
	Service string    `json:"service"`
	Version string    `json:"version"`
	CPA     CPAInfo   `json:"cpa"`
	Usage   UsageInfo `json:"usage"`
}

type CPAInfo struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type UsageInfo struct {
	Enabled bool `json:"enabled"`
}

func RegisterRoutes(engine *gin.Engine, info Info) {
	if engine == nil {
		return
	}
	if info.Service == "" {
		info.Service = "cpa-pc"
	}
	if info.Version == "" {
		info.Version = "dev"
	}

	engine.GET("/cpa-pc/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, info)
	})
}
