package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	"github.com/HWliao/CPA-PC/internal/httpapi"
	pcstore "github.com/HWliao/CPA-PC/internal/store"
	pcusage "github.com/HWliao/CPA-PC/internal/usage"
	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/api"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/api/handlers"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy"
	cpaconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
	_ "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator/builtin"
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
	if err := ensureEmbeddedWritablePath(cfg); err != nil {
		return nil, err
	}
	logCleanup, err := configureEmbeddedLogOutput(cfg)
	if err != nil {
		return nil, err
	}
	configureEmbeddedLogLevel(cpaCfg.Debug)
	cleanupFns := make([]func() error, 0, 2)
	if logCleanup != nil {
		cleanupFns = append(cleanupFns, logCleanup)
	}
	var usageStore *pcstore.Store
	if cfg.Usage.Enabled {
		usageStore, err = pcstore.Open(cfg.Runtime.UsageDBPath)
		if err != nil {
			closeCleanupFns(cleanupFns)
			return nil, fmt.Errorf("open usage store: %w", err)
		}
		cleanupFns = append(cleanupFns, usageStore.Close)
	}

	info := httpapi.Info{
		Version:   opts.Version,
		BuildDate: opts.BuildDate,
		CLIProxyAPI: httpapi.CLIProxyAPIInfo{
			Version: resolveCLIProxyAPIVersion(),
		},
		CPA: httpapi.CPAInfo{
			Host: cpaCfg.Host,
			Port: cpaCfg.Port,
		},
		Usage: httpapi.UsageInfo{
			Enabled: cfg.Usage.Enabled,
		},
	}
	var routeStore httpapi.UsageStore
	if usageStore != nil {
		routeStore = usageStore
	}

	service, err := cliproxy.NewBuilder().
		WithConfig(cpaCfg).
		WithConfigPath(cfg.Runtime.ConfigPath).
		WithServerOptions(api.WithMiddleware(buildInfoHeaderMiddleware(opts.BuildDate))).
		WithServerOptions(api.WithRouterConfigurator(func(engine *gin.Engine, handler *handlers.BaseAPIHandler, _ *cpaconfig.Config) {
			httpapi.RegisterRoutesWithOptions(engine, httpapi.RouteOptions{
				Info:                      info,
				Store:                     routeStore,
				Config:                    cfg,
				ManagementKey:             cfg.RemoteManagement.SecretKey,
				StartedAt:                 time.Now(),
				ChartAuthMetadataProvider: chartAuthMetadataProvider(handler),
			})
		})).
		Build()
	if err != nil {
		closeCleanupFns(cleanupFns)
		return nil, fmt.Errorf("build embedded CPA service: %w", err)
	}
	proxyService := ProxyService(service)

	if usageStore != nil {
		writer := opts.Stdout
		if writer == nil {
			writer = os.Stdout
		}
		service.RegisterUsagePlugin(pcusage.NewPersistPlugin(usageStore, writer))
	}

	if len(cleanupFns) > 0 {
		proxyService = &serviceWithCleanup{service: proxyService, cleanups: cleanupFns}
	}

	return proxyService, nil
}

func chartAuthMetadataProvider(handler *handlers.BaseAPIHandler) func(context.Context) []pcusage.ChartAuthMetadata {
	return func(context.Context) []pcusage.ChartAuthMetadata {
		return chartAuthMetadataFromHandler(handler)
	}
}

func chartAuthMetadataFromHandler(handler *handlers.BaseAPIHandler) []pcusage.ChartAuthMetadata {
	if handler == nil || handler.AuthManager == nil {
		return nil
	}
	auths := handler.AuthManager.List()
	items := make([]pcusage.ChartAuthMetadata, 0, len(auths))
	for _, auth := range auths {
		if auth == nil {
			continue
		}
		authIndex := strings.TrimSpace(auth.EnsureIndex())
		if authIndex == "" {
			continue
		}
		authFile := strings.TrimSpace(auth.FileName)
		if authFile == "" {
			authFile = strings.TrimSpace(auth.ID)
		}
		accountType, account := auth.AccountInfo()
		account = strings.TrimSpace(account)
		if strings.EqualFold(accountType, "api_key") {
			account = ""
		}
		label := firstNonEmptyString(auth.Label, authFile, account, authIndex)
		if account == "" {
			account = label
		}
		items = append(items, pcusage.ChartAuthMetadata{
			AuthID:    strings.TrimSpace(auth.ID),
			AuthIndex: authIndex,
			Account:   account,
			Label:     label,
			AuthFile:  authFile,
		})
	}
	return items
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type serviceWithCleanup struct {
	service  ProxyService
	cleanups []func() error
}

func (s *serviceWithCleanup) Run(ctx context.Context) error {
	defer closeCleanupFns(s.cleanups)
	return s.service.Run(ctx)
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
