package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigFileName  = "config.yaml"
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

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", resolvedConfigPath, err)
	}

	cfg.applyDefaults()
	if err := cfg.resolveRuntimePaths(resolvedConfigPath); err != nil {
		return nil, err
	}

	return &cfg, nil
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
