package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
	"github.com/gin-gonic/gin"
	sdkapi "github.com/router-for-me/CLIProxyAPI/v8/sdk/api"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v8/sdk/config"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type ConfigMigrationPreview struct {
	ConfigPath         string
	CompanionPath      string
	BackupPath         string
	AlreadyV8          bool
	UnknownSections    []string
	SDKSettingsChanged bool
	ScopeChanges       []string
}

type preparedMigration struct {
	preview   ConfigMigrationPreview
	original  []byte
	v8Config  []byte
	companion []byte
	before    *pcconfig.Config
}

func PreviewConfigMigration(configPath string) (ConfigMigrationPreview, error) {
	prepared, err := prepareConfigMigration(configPath)
	if err != nil {
		return ConfigMigrationPreview{}, err
	}
	return prepared.preview, nil
}

func MigrateConfig(configPath string) (ConfigMigrationPreview, error) {
	prepared, err := prepareConfigMigration(configPath)
	if err != nil {
		return ConfigMigrationPreview{}, err
	}
	if prepared.preview.AlreadyV8 {
		return prepared.preview, nil
	}
	if len(prepared.preview.UnknownSections) > 0 {
		return ConfigMigrationPreview{}, fmt.Errorf("unmapped SDK config sections require review: %s", strings.Join(prepared.preview.UnknownSections, ", "))
	}
	if prepared.preview.SDKSettingsChanged {
		return ConfigMigrationPreview{}, fmt.Errorf("SDK settings differ after v8 conversion; review the source config before migration")
	}
	if err := writeNewMigrationFile(prepared.preview.BackupPath, prepared.original); err != nil {
		return ConfigMigrationPreview{}, fmt.Errorf("back up config: %w", err)
	}
	if err := writeNewMigrationFile(prepared.preview.CompanionPath, prepared.companion); err != nil {
		_ = os.Remove(prepared.preview.BackupPath)
		return ConfigMigrationPreview{}, fmt.Errorf("write CPA-PC companion: %w", err)
	}
	if err := sdkapi.WriteConfig(prepared.preview.ConfigPath, prepared.v8Config); err != nil {
		return ConfigMigrationPreview{}, rollbackMigration(prepared, fmt.Errorf("write SDK v8 config: %w", err))
	}
	after, err := pcconfig.Load(prepared.preview.ConfigPath)
	if err == nil {
		if after.Host != prepared.before.Host || after.Port != prepared.before.Port ||
			after.Runtime.AuthDir != prepared.before.Runtime.AuthDir ||
			after.Runtime.DataDir != prepared.before.Runtime.DataDir ||
			after.Runtime.LogsDir != prepared.before.Runtime.LogsDir ||
			after.Runtime.StaticDir != prepared.before.Runtime.StaticDir ||
			after.Runtime.UsageDBPath != prepared.before.Runtime.UsageDBPath ||
			after.Usage != prepared.before.Usage ||
			after.LoggingToFile != prepared.before.LoggingToFile ||
			after.RemoteManagement.AllowRemote != prepared.before.RemoteManagement.AllowRemote ||
			after.RemoteManagement.DisableControlPanel != prepared.before.RemoteManagement.DisableControlPanel ||
			(after.RemoteManagement.SecretKey != prepared.before.RemoteManagement.SecretKey &&
				bcrypt.CompareHashAndPassword([]byte(after.RemoteManagement.SecretKey), []byte(prepared.before.RemoteManagement.SecretKey)) != nil) {
			err = fmt.Errorf("migrated settings do not match the original")
		}
	}
	if err != nil {
		return ConfigMigrationPreview{}, rollbackMigration(prepared, fmt.Errorf("validate migrated config: %w", err))
	}
	return prepared.preview, nil
}

func rollbackMigration(prepared *preparedMigration, cause error) error {
	if err := os.WriteFile(prepared.preview.ConfigPath, prepared.original, 0o600); err != nil {
		return fmt.Errorf("%w; restore config from %s: %v", cause, prepared.preview.BackupPath, err)
	}
	if err := os.Remove(prepared.preview.CompanionPath); err != nil {
		return fmt.Errorf("%w; remove companion %s: %v", cause, prepared.preview.CompanionPath, err)
	}
	return cause
}

func prepareConfigMigration(configPath string) (*preparedMigration, error) {
	resolved, err := pcconfig.ResolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}
	original, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	preview := ConfigMigrationPreview{
		ConfigPath:    resolved,
		CompanionPath: filepath.Join(filepath.Dir(resolved), pcconfig.CPAConfigFileName),
		BackupPath:    resolved + ".v0.bak",
	}
	v8, err := pcconfig.IsV8Config(original)
	if err != nil {
		return nil, err
	}
	if v8 {
		if _, err := pcconfig.Load(resolved); err != nil {
			return nil, err
		}
		preview.AlreadyV8 = true
		return &preparedMigration{preview: preview}, nil
	}
	for _, path := range []string{preview.CompanionPath, preview.BackupPath} {
		if _, err := os.Stat(path); err == nil {
			return nil, fmt.Errorf("migration target already exists: %s", path)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	before, err := pcconfig.Load(resolved)
	if err != nil {
		return nil, err
	}
	sdkCfg, err := sdkconfig.ParseConfigBytes(original)
	if err != nil {
		return nil, fmt.Errorf("parse SDK config: %w", err)
	}
	secret := sdkCfg.RemoteManagement.SecretKey
	if secret == "" {
		hash, errHash := bcrypt.GenerateFromPassword([]byte(pcconfig.DefaultManagementKey), bcrypt.DefaultCost)
		if errHash != nil {
			return nil, errHash
		}
		secret = string(hash)
	}

	handler := sdkapi.NewHandler(sdkCfg, resolved, nil)
	engine := gin.New()
	engine.GET("/v8/management/config.yaml", handler.ConfigV8)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v8/management/config.yaml", nil))
	if response.Code != http.StatusOK {
		return nil, fmt.Errorf("SDK v8 config preview failed with status %d", response.Code)
	}
	preview.UnknownSections = unmappedSections(original, response.Body.Bytes())
	v8Config, err := setManagementSecret(response.Body.Bytes(), secret)
	if err != nil {
		return nil, err
	}
	converted, err := sdkconfig.ParseConfigBytes(v8Config)
	if err != nil {
		return nil, fmt.Errorf("validate SDK v8 config: %w", err)
	}
	for key, value := range converted.OAuthOnlyFields {
		if sdkCfg.OAuthOnlyFields[key] != value {
			preview.ScopeChanges = append(preview.ScopeChanges, key)
		}
	}
	sort.Strings(preview.ScopeChanges)
	sdkCfg.OAuthOnlyFields = converted.OAuthOnlyFields
	sdkCfg.RemoteManagement.SecretKey = converted.RemoteManagement.SecretKey
	oldPlugins, err := json.Marshal(sdkCfg.Plugins.Configs)
	if err != nil {
		return nil, err
	}
	newPlugins, err := json.Marshal(converted.Plugins.Configs)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(oldPlugins, newPlugins) {
		sdkCfg.Plugins.Configs = converted.Plugins.Configs
	}
	preview.SDKSettingsChanged = !reflect.DeepEqual(sdkCfg, converted)
	companion, err := yaml.Marshal(struct {
		DataDir   string         `yaml:"data-dir"`
		LogsDir   string         `yaml:"logs-dir"`
		StaticDir string         `yaml:"static-dir"`
		Usage     pcconfig.Usage `yaml:"usage"`
	}{before.DataDir, before.LogsDir, before.StaticDir, before.Usage})
	if err != nil {
		return nil, err
	}
	return &preparedMigration{preview: preview, original: original, v8Config: v8Config, companion: companion, before: before}, nil
}

func unmappedSections(original, migrated []byte) []string {
	var oldDoc yaml.Node
	if yaml.Unmarshal(original, &oldDoc) != nil || len(oldDoc.Content) == 0 {
		return nil
	}
	knownSDKRoots := make(map[string]bool)
	for _, typ := range []reflect.Type{reflect.TypeOf(sdkconfig.Config{}), reflect.TypeOf(sdkconfig.SDKConfig{})} {
		for i := 0; i < typ.NumField(); i++ {
			name, _, _ := strings.Cut(typ.Field(i).Tag.Get("yaml"), ",")
			if name != "" && name != "-" {
				knownSDKRoots[name] = true
			}
		}
	}
	comments := strings.Split(string(migrated), "\n")
	var unknown []string
	for i := 0; i+1 < len(oldDoc.Content[0].Content); i += 2 {
		key := oldDoc.Content[0].Content[i].Value
		if knownSDKRoots[key] {
			continue
		}
		switch key {
		case "data-dir", "logs-dir", "static-dir", "usage":
			continue
		}
		for _, line := range comments {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# "+key+":") || strings.HasPrefix(line, "# "+key+".") {
				unknown = append(unknown, key)
				break
			}
		}
	}
	return unknown
}

func setManagementSecret(data []byte, secret string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	root := doc.Content[0]
	management := mappingValue(root, "management")
	if management == nil {
		management = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "management"}, management)
	}
	key := mappingValue(management, "secret-key")
	if key == nil {
		key = &yaml.Node{}
		management.Content = append(management.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "secret-key"}, key)
	}
	key.Kind, key.Tag, key.Value = yaml.ScalarNode, "!!str", secret
	return yaml.Marshal(&doc)
}

func mappingValue(node *yaml.Node, name string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == name {
			return node.Content[i+1]
		}
	}
	return nil
}

func writeNewMigrationFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
	}
	return err
}
