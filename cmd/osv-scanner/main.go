package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/osv-scanner/internal/output"
	"github.com/google/osv-scanner/pkg/osvscanner"

	"github.com/urfave/cli/v2"
)

var (
	version = "dev"
	commit  = "n/a"
	date    = "n/a"
)

func run(args []string, stdout, stderr io.Writer) int {
	var r *output.Reporter

	cli.VersionPrinter = func(ctx *cli.Context) {
		r = output.NewReporter(stdout, stderr, "")
		r.PrintText(fmt.Sprintf("osv-scanner version: %s\ncommit: %s\nbuilt at: %s\n", ctx.App.Version, commit, date))
	}

	app := &cli.App{
		Name:      "osv-scanner",
		Version:   version,
		Usage:     "scans various mediums for dependencies and matches it against the OSV database",
		Suggest:   true,
		Writer:    stdout,
		ErrWriter: stderr,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:      "docker",
				Aliases:   []string{"D"},
				Usage:     "scan docker image with this name",
				TakesFile: false,
			},
			&cli.StringSliceFlag{
				Name:      "lockfile",
				Aliases:   []string{"L"},
				Usage:     "scan package lockfile on this path",
				TakesFile: true,
			},
			&cli.StringSliceFlag{
				Name:      "sbom",
				Aliases:   []string{"S"},
				Usage:     "scan sbom file on this path",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:      "config",
				Usage:     "set/override config file",
				TakesFile: true,
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "sets the output format",
				Value:   "table",
				Action: func(context *cli.Context, s string) error {
					if s != "table" && s != "json" {
						return fmt.Errorf("unsupported output format \"%s\" - must be either \"table\" or \"json\"", s)
					}

					return nil
				},
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "sets output to json (deprecated, use --format json instead)",
			},
			&cli.BoolFlag{
				Name:  "skip-git",
				Usage: "skip scanning git repositories",
				Value: false,
			},
			&cli.BoolFlag{
				Name:    "recursive",
				Aliases: []string{"r"},
				Usage:   "check subdirectories",
				Value:   false,
			},
		},
		ArgsUsage: "[directory1 directory2...]",
		Action: func(context *cli.Context) error {
			format := context.String("format")

			if context.Bool("json") {
				format = "json"
			}

			r = output.NewReporter(stdout, stderr, format)

			vulnResult, err := osvscanner.DoScan(osvscanner.ScannerActions{
				LockfilePaths:        context.StringSlice("lockfile"),
				SBOMPaths:            context.StringSlice("sbom"),
				DockerContainerNames: context.StringSlice("docker"),
				Recursive:            context.Bool("recursive"),
				SkipGit:              context.Bool("skip-git"),
				ConfigOverridePath:   context.String("config"),
				DirectoryPaths:       context.Args().Slice(),
			}, r)

			if errPrint := r.PrintResult(&vulnResult); errPrint != nil {
				return fmt.Errorf("failed to write output: %w", errPrint)
			}
			//nolint:wrapcheck
			return err
		},
	}

	if err := app.Run(args); err != nil {
		if r == nil {
			r = output.NewReporter(stdout, stderr, "")
		}
		if errors.Is(err, osvscanner.VulnerabilitiesFoundErr) {
			return 1
		}

		if errors.Is(err, osvscanner.NoPackagesFoundErr) {
			r.PrintError("No package sources found, --help for usage information.\n")
			return 128
		}

		r.PrintError(fmt.Sprintf("%v\n", err))

		return 127
	}

	return 0
}

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}
