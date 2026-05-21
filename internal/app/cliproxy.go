package app

import (
	"fmt"
	"os"
	"strings"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	"github.com/HWliao/CPA-PC/internal/httpapi"
	pcusage "github.com/HWliao/CPA-PC/internal/usage"
	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/api"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/api/handlers"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy"
	cpaconfig "github.com/router-for-me/CLIProxyAPI/v6/sdk/config"
	"golang.org/x/crypto/bcrypt"
)

func NewCLIProxyService(cfg *pcconfig.Config, opts ServiceOptions) (ProxyService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cpa-pc: config is required")
	}

	cpaCfg, err := loadCPAConfig(cfg)
	if err != nil {
		return nil, err
	}
	if err := configureManagementStaticPath(cfg.Runtime.StaticDir); err != nil {
		return nil, err
	}

	info := httpapi.Info{
		Version: opts.Version,
		CPA: httpapi.CPAInfo{
			Host: cpaCfg.Host,
			Port: cpaCfg.Port,
		},
		Usage: httpapi.UsageInfo{
			Enabled: cfg.Usage.Enabled,
		},
	}

	service, err := cliproxy.NewBuilder().
		WithConfig(cpaCfg).
		WithConfigPath(cfg.Runtime.ConfigPath).
		WithServerOptions(api.WithRouterConfigurator(func(engine *gin.Engine, _ *handlers.BaseAPIHandler, _ *cpaconfig.Config) {
			httpapi.RegisterRoutes(engine, info)
		})).
		Build()
	if err != nil {
		return nil, fmt.Errorf("build embedded CPA service: %w", err)
	}

	if cfg.Usage.Enabled {
		writer := opts.Stdout
		if writer == nil {
			writer = os.Stdout
		}
		service.RegisterUsagePlugin(pcusage.NewLogPlugin(writer))
	}

	return service, nil
}

func loadCPAConfig(cfg *pcconfig.Config) (*cpaconfig.Config, error) {
	cpaCfg, err := cpaconfig.LoadConfig(cfg.Runtime.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load CPA config: %w", err)
	}

	cpaCfg.Host = cfg.Host
	cpaCfg.Port = cfg.Port
	cpaCfg.AuthDir = cfg.Runtime.AuthDir
	cpaCfg.LoggingToFile = cfg.LoggingToFile
	cpaCfg.RemoteManagement.AllowRemote = cfg.RemoteManagement.AllowRemote
	cpaCfg.RemoteManagement.DisableControlPanel = cfg.RemoteManagement.DisableControlPanel

	if cpaCfg.RemoteManagement.SecretKey == "" && cfg.RemoteManagement.SecretKey != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.RemoteManagement.SecretKey), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash default management key: %w", err)
		}
		cpaCfg.RemoteManagement.SecretKey = string(hashed)
	}

	return cpaCfg, nil
}

func configureManagementStaticPath(staticDir string) error {
	if strings.TrimSpace(os.Getenv("MANAGEMENT_STATIC_PATH")) != "" {
		return nil
	}
	staticDir = strings.TrimSpace(staticDir)
	if staticDir == "" {
		return nil
	}
	if err := os.Setenv("MANAGEMENT_STATIC_PATH", staticDir); err != nil {
		return fmt.Errorf("set management static path: %w", err)
	}
	return nil
}
