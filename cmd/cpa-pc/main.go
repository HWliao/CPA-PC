package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/HWliao/CPA-PC/internal/app"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("cpa-pc", flag.ContinueOnError)
	flags.SetOutput(stderr)

	configPath := flags.String("config", "", "path to config.yaml; defaults to config.yaml next to cpa-pc.exe")
	showVersion := flags.Bool("version", false, "print version and exit")

	flags.Usage = func() {
		fmt.Fprintln(flags.Output(), "Usage of cpa-pc:")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "cpa-pc %s\n", version)
		return 0
	}

	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %v\n", flags.Args())
		flags.Usage()
		return 2
	}

	if err := app.Run(ctx, app.Options{
		ConfigPath: *configPath,
		Version:    version,
		BuildDate:  buildDate,
		Stdout:     stdout,
	}); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	return 0
}
