package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
)

func TestRunLoadsConfigPrintsSummaryAndRunsProxy(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("host: 127.0.0.1\nport: 8317\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	proxy := &fakeProxyService{}
	err := Run(context.Background(), Options{
		ConfigPath: configPath,
		Version:    "test",
		BuildDate:  "2026-05-22T00:00:00Z",
		Stdout:     &stdout,
		ProxyFactory: func(cfg *pcconfig.Config, opts ServiceOptions) (ProxyService, error) {
			if cfg.Runtime.ConfigPath != filepath.Clean(configPath) {
				t.Fatalf("ConfigPath = %q, want %q", cfg.Runtime.ConfigPath, filepath.Clean(configPath))
			}
			if opts.Version != "test" {
				t.Fatalf("Version = %q, want test", opts.Version)
			}
			if opts.BuildDate != "2026-05-22T00:00:00Z" {
				t.Fatalf("BuildDate = %q, want 2026-05-22T00:00:00Z", opts.BuildDate)
			}
			return proxy, nil
		},
	})

	if err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "CPA-PC test foundation initialized") {
		t.Fatalf("stdout = %q, want foundation summary", output)
	}
	if !strings.Contains(output, filepath.Clean(configPath)) {
		t.Fatalf("stdout = %q, want config path", output)
	}
	if !proxy.ran {
		t.Fatal("proxy service was not run")
	}
}

func TestRunTreatsContextCancellationAsCleanShutdown(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := Run(context.Background(), Options{
		ConfigPath: configPath,
		ProxyFactory: func(_ *pcconfig.Config, _ ServiceOptions) (ProxyService, error) {
			return &fakeProxyService{err: context.Canceled}, nil
		},
	})

	if err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
}

func TestRunReturnsProxyErrors(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("port: 8317\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wantErr := errors.New("proxy failed")

	err := Run(context.Background(), Options{
		ConfigPath: configPath,
		ProxyFactory: func(_ *pcconfig.Config, _ ServiceOptions) (ProxyService, error) {
			return &fakeProxyService{err: wantErr}, nil
		},
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Run returned %v, want %v", err, wantErr)
	}
}

type fakeProxyService struct {
	ran bool
	err error
}

func (s *fakeProxyService) Run(context.Context) error {
	s.ran = true
	return s.err
}
