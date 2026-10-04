package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFileName  = "config.yaml"
	CPAConfigFileName      = "cpa-pc.yaml"
	DefaultHost            = ""
	DefaultPort            = 8317
	DefaultManagementKey   = "123456"
	DefaultDataDir         = "./data"
	DefaultLogsDir         = "./logs"
	DefaultStaticDir       = "./static"
	DefaultAuthDir         = "~/.cli-proxy-api"
	DefaultUsageDBPath     = "./data/usage.sqlite"
	DefaultUsageQueryLimit = 50000
)

type Config struct {
	Host             string           `yaml:"host"`
	Port             int              `yaml:"port"`
	DataDir          string           `yaml:"data-dir"`
	LogsDir          string           `yaml:"logs-dir"`
	StaticDir        string           `yaml:"static-dir"`
	RemoteManagement RemoteManagement `yaml:"remote-management"`
	AuthDir          string           `yaml:"auth-dir"`
	APIKeys          []string         `yaml:"api-keys"`
	Usage            Usage            `yaml:"usage"`
	LoggingToFile    bool             `yaml:"logging-to-file"`
	Runtime          RuntimePaths     `yaml:"-"`
}

type RemoteManagement struct {
	AllowRemote         bool   `yaml:"allow-remote"`
	SecretKey           string `yaml:"secret-key"`
	DisableControlPanel bool   `yaml:"disable-control-panel"`
}

type Usage struct {
	Enabled    bool   `yaml:"enabled"`
	DBPath     string `yaml:"db-path"`
	QueryLimit int    `yaml:"query-limit"`
}

type RuntimePaths struct {
	BaseDir     string
	ConfigPath  string
	DataDir     string
	LogsDir     string
	StaticDir   string
	AuthDir     string
	UsageDBPath string
}

type MissingConfigError struct {
	ConfigPath string
}

func (err MissingConfigError) Error() string {
	return fmt.Sprintf("config file not found: %s; copy config.example.yaml to config.yaml or pass -config <path>", err.ConfigPath)
}

func Default() Config {
	return Config{
		Host:      DefaultHost,
		Port:      DefaultPort,
		DataDir:   DefaultDataDir,
		LogsDir:   DefaultLogsDir,
		StaticDir: DefaultStaticDir,
		RemoteManagement: RemoteManagement{
			AllowRemote:         false,
			SecretKey:           DefaultManagementKey,
			DisableControlPanel: false,
		},
		AuthDir:       DefaultAuthDir,
		Usage:         Usage{Enabled: true, DBPath: DefaultUsageDBPath, QueryLimit: DefaultUsageQueryLimit},
		LoggingToFile: false,
	}
}

func Load(configPath string) (*Config, error) {
	resolvedConfigPath, err := ResolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolvedConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, MissingConfigError{ConfigPath: resolvedConfigPath}
		}
		return nil, fmt.Errorf("read config file %s: %w", resolvedConfigPath, err)
	}

	v8, err := IsV8Config(data)
	if err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", resolvedConfigPath, err)
	}
	cfg := Default()
	managementKeyExplicitlyEmpty := false
	if v8 {
		sdkCfg, err := sdkconfig.ParseConfigBytes(data)
		if err != nil {
			return nil, fmt.Errorf("parse SDK config file %s: %w", resolvedConfigPath, err)
		}
		cfg.Host = sdkCfg.Host
		cfg.Port = sdkCfg.Port
		cfg.AuthDir = sdkCfg.AuthDir
		cfg.APIKeys = sdkCfg.APIKeys
		cfg.LoggingToFile = sdkCfg.LoggingToFile
		cfg.RemoteManagement.AllowRemote = sdkCfg.RemoteManagement.AllowRemote
		cfg.RemoteManagement.DisableControlPanel = sdkCfg.RemoteManagement.DisableControlPanel
		if sdkCfg.RemoteManagement.SecretKey != "" {
			cfg.RemoteManagement.SecretKey = sdkCfg.RemoteManagement.SecretKey
		} else {
			var doc yaml.Node
			if err := yaml.Unmarshal(data, &doc); err == nil && len(doc.Content) > 0 {
				for i := 0; i+1 < len(doc.Content[0].Content); i += 2 {
					if doc.Content[0].Content[i].Value == "management" {
						management := doc.Content[0].Content[i+1]
						for j := 0; j+1 < len(management.Content); j += 2 {
							if management.Content[j].Value == "secret-key" {
								managementKeyExplicitlyEmpty = true
							}
						}
					}
				}
			}
		}
	} else if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", resolvedConfigPath, err)
	}

	companionName := CPAConfigFileName
	if filepath.Base(resolvedConfigPath) == "config.example.yaml" {
		companionName = "cpa-pc.example.yaml"
	}
	companionPath := filepath.Join(filepath.Dir(resolvedConfigPath), companionName)
	companion, err := os.ReadFile(companionPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read CPA-PC config file %s: %w", companionPath, err)
		}
		if v8 {
			return nil, fmt.Errorf("v8 config requires %s; migrate CPA-PC settings before using v8 config", companionPath)
		}
	} else {
		local := struct {
			DataDir   string `yaml:"data-dir"`
			LogsDir   string `yaml:"logs-dir"`
			StaticDir string `yaml:"static-dir"`
			Usage     Usage  `yaml:"usage"`
		}{cfg.DataDir, cfg.LogsDir, cfg.StaticDir, cfg.Usage}
		decoder := yaml.NewDecoder(bytes.NewReader(companion))
		decoder.KnownFields(true)
		if err := decoder.Decode(&local); err != nil {
			return nil, fmt.Errorf("parse CPA-PC config file %s: %w", companionPath, err)
		}
		cfg.DataDir, cfg.LogsDir, cfg.StaticDir, cfg.Usage = local.DataDir, local.LogsDir, local.StaticDir, local.Usage
	}

	cfg.applyDefaults()
	if managementKeyExplicitlyEmpty {
		cfg.RemoteManagement.SecretKey = ""
	}
	if err := cfg.resolveRuntimePaths(resolvedConfigPath); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IsV8Config reports whether a YAML document contains SDK v8 configuration fields.
func IsV8Config(data []byte) (bool, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return false, nil
	}
	root := doc.Content[0]
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i].Value, root.Content[i+1]
		switch key {
		case "config-version", "server", "management", "access", "oauth", "observability", "upstream", "credentials", "requests", "multimedia":
			return true, nil
		case "api-keys":
			if value.Kind == yaml.MappingNode {
				return true, nil
			}
		case "routing":
			if value.Kind != yaml.MappingNode {
				continue
			}
			for j := 0; j+1 < len(value.Content); j += 2 {
				switch value.Content[j].Value {
				case "retry", "cooldown", "force-model-prefix":
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (cfg Config) ListenAddress() string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

func ResolveConfigPath(configPath string) (string, error) {
	if configPath == "" {
		executablePath, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve executable path: %w", err)
		}
		return filepath.Clean(filepath.Join(filepath.Dir(executablePath), DefaultConfigFileName)), nil
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("resolve config path %s: %w", configPath, err)
	}

	return filepath.Clean(absPath), nil
}

func (cfg *Config) applyDefaults() {
	if cfg.Host == "" {
		cfg.Host = DefaultHost
	}
	if cfg.Port <= 0 {
		cfg.Port = DefaultPort
	}
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultDataDir
	}
	if cfg.LogsDir == "" {
		cfg.LogsDir = DefaultLogsDir
	}
	if cfg.StaticDir == "" {
		cfg.StaticDir = DefaultStaticDir
	}
	if cfg.RemoteManagement.SecretKey == "" {
		cfg.RemoteManagement.SecretKey = DefaultManagementKey
	}
	if cfg.AuthDir == "" {
		cfg.AuthDir = DefaultAuthDir
	}
	if cfg.Usage.DBPath == "" {
		cfg.Usage.DBPath = DefaultUsageDBPath
	}
	if cfg.Usage.QueryLimit <= 0 {
		cfg.Usage.QueryLimit = DefaultUsageQueryLimit
	}
}

func (cfg *Config) resolveRuntimePaths(configPath string) error {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return fmt.Errorf("resolve config path %s: %w", configPath, err)
	}

	baseDir := filepath.Dir(absConfigPath)
	cfg.Runtime = RuntimePaths{
		BaseDir:     baseDir,
		ConfigPath:  filepath.Clean(absConfigPath),
		DataDir:     resolvePath(baseDir, cfg.DataDir),
		LogsDir:     resolvePath(baseDir, cfg.LogsDir),
		StaticDir:   resolvePath(baseDir, cfg.StaticDir),
		AuthDir:     resolvePathWithHome(baseDir, cfg.AuthDir),
		UsageDBPath: resolvePath(baseDir, cfg.Usage.DBPath),
	}

	return nil
}

func resolvePath(baseDir string, value string) string {
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(baseDir, value))
}

func resolvePathWithHome(baseDir string, value string) string {
	if value == "" || value[0] != '~' {
		return resolvePath(baseDir, value)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean(value)
	}

	remainder := strings.TrimLeft(value[1:], `/\`)
	if remainder == "" {
		return filepath.Clean(homeDir)
	}

	return filepath.Clean(filepath.Join(homeDir, filepath.FromSlash(remainder)))
}
