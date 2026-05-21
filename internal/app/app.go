package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
)

type ProxyService interface {
	Run(context.Context) error
}

type ProxyFactory func(*pcconfig.Config, ServiceOptions) (ProxyService, error)

type ServiceOptions struct {
	Version string
	Stdout  io.Writer
}

type Options struct {
	ConfigPath   string
	Version      string
	Stdout       io.Writer
	ProxyFactory ProxyFactory
}

func Run(ctx context.Context, opts Options) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	cfg, err := pcconfig.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	stdout := opts.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	version := opts.Version
	if version == "" {
		version = "dev"
	}

	fmt.Fprintf(stdout, "CPA-PC %s foundation initialized\n", version)
	fmt.Fprintf(stdout, "config: %s\n", cfg.Runtime.ConfigPath)
	fmt.Fprintf(stdout, "listen: %s\n", cfg.ListenAddress())
	fmt.Fprintf(stdout, "usage database: %s\n", cfg.Runtime.UsageDBPath)

	factory := opts.ProxyFactory
	if factory == nil {
		factory = NewCLIProxyService
	}

	service, err := factory(cfg, ServiceOptions{Version: version, Stdout: stdout})
	if err != nil {
		return err
	}

	if err := service.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
