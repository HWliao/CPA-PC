package app

import (
	"context"
	"fmt"
	"io"

	pcconfig "github.com/HWliao/CPA-PC/internal/config"
)

type Options struct {
	ConfigPath string
	Version    string
	Stdout     io.Writer
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

	return nil
}
