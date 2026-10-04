package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

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
	previewMigration := flags.Bool("preview-config-migration", false, "preview a v0 to v8 config migration without writing files")
	migrateConfig := flags.Bool("migrate-config", false, "back up config.yaml and migrate it to v8 with cpa-pc.yaml")

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
	if *previewMigration && *migrateConfig {
		fmt.Fprintln(stderr, "choose either -preview-config-migration or -migrate-config")
		return 2
	}
	if *previewMigration || *migrateConfig {
		var preview app.ConfigMigrationPreview
		var err error
		if *migrateConfig {
			preview, err = app.MigrateConfig(*configPath)
		} else {
			preview, err = app.PreviewConfigMigration(*configPath)
		}
		if err != nil {
			fmt.Fprintf(stderr, "config migration: %v\n", err)
			return 1
		}
		if preview.AlreadyV8 {
			fmt.Fprintf(stdout, "config already uses v8: %s\n", preview.ConfigPath)
		} else {
			fmt.Fprintf(stdout, "config: %s\nCPA-PC settings: %s\nbackup: %s\n", preview.ConfigPath, preview.CompanionPath, preview.BackupPath)
			if len(preview.UnknownSections) > 0 {
				fmt.Fprintf(stdout, "unmapped SDK sections requiring review: %s\n", strings.Join(preview.UnknownSections, ", "))
			}
			if preview.SDKSettingsChanged {
				fmt.Fprintln(stdout, "SDK settings would change; review the source config before migration")
			}
			if len(preview.ScopeChanges) > 0 {
				fmt.Fprintf(stdout, "SDK OAuth scope changes to review: %s\n", strings.Join(preview.ScopeChanges, ", "))
			}
			if *previewMigration {
				fmt.Fprintln(stdout, "run -migrate-config to apply after reviewing the paths")
			} else {
				fmt.Fprintln(stdout, "config migrated to v8")
			}
		}
		return 0
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
