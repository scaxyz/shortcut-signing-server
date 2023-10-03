package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/scaxyz/shortcut-signing-server/internal"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
	"golang.org/x/exp/slices"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

var logLevels = []string{"debug", "info", "warn", "error", "fatal", "panic", "trace"}
var logFormats = []string{"text", "json"}
var logFile io.WriteCloser

const (
	FLAG_CATEGORY_LOGGING = "LOGGING"
	FLAG_CATEGORY_SERVER  = "SERVER"
)

func main() {

	var globalFlags []cli.Flag
	var serveFlags []cli.Flag

	configFlag := &cli.PathFlag{
		Name:  "config",
		Value: "",
		Usage: "Path to the yaml config file",
		// Category: FLAG_CATEGORY_SERVER,
	}

	tlsFlags := []cli.Flag{
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:     "tls",
			Value:    false,
			Usage:    "Enable TLS",
			Category: FLAG_CATEGORY_SERVER,
		}),

		altsrc.NewPathFlag(&cli.PathFlag{
			Name:     "tls-cert",
			Value:    "",
			Usage:    "Path to the tls cert file",
			Category: FLAG_CATEGORY_SERVER,
		}),
		altsrc.NewPathFlag(&cli.PathFlag{
			Name:     "tls-key",
			Value:    "",
			Usage:    "Path to the tls cert key",
			Category: FLAG_CATEGORY_SERVER,
		}),
	}

	loggingFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "log-level",
			Value:    "info",
			Usage:    fmt.Sprintf("Available log levels: %s", strings.Join(logLevels, ", ")),
			Category: FLAG_CATEGORY_LOGGING,
			Action: func(ctx *cli.Context, logLevel string) error {
				if slices.Contains(logLevels, logLevel) {
					return nil
				}
				return fmt.Errorf("'%s' is not a valid log level", logLevel)
			},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "log-format",
			Value:    "text",
			Usage:    fmt.Sprintf("Available log formats: %s", strings.Join(logFormats, ", ")),
			Category: FLAG_CATEGORY_LOGGING,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:     "log-file",
			Value:    "",
			Usage:    "Log file path",
			Category: FLAG_CATEGORY_LOGGING,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:     "debug",
			Value:    false,
			Usage:    "Set log level to debug",
			Category: FLAG_CATEGORY_LOGGING,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:     "quiet",
			Aliases:  []string{"q"},
			Usage:    "Disable logging to stdout",
			Value:    false,
			Category: FLAG_CATEGORY_LOGGING,
		}),
	}

	serverFlags := []cli.Flag{
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:     "real-error-responses",
			Value:    false,
			Usage:    "Return real errors instead of only HTTP codes names in the response",
			Category: FLAG_CATEGORY_SERVER,
			Aliases:  []string{"re"},
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:     "max-concurrent-jobs",
			Value:    0,
			Usage:    "Maximum number of concurrent signing jobs",
			Category: FLAG_CATEGORY_SERVER,
		}),
		altsrc.NewPathFlag(&cli.PathFlag{
			Name:     "templates",
			Value:    "",
			Usage:    "Folder containing custom html templates (currently only: 'form.html' is used)",
			Category: FLAG_CATEGORY_SERVER,
		}),
	}

	globalFlags = append(globalFlags, loggingFlags...)
	globalFlags = append(globalFlags, configFlag)

	serveFlags = append(serveFlags, tlsFlags...)
	serveFlags = append(serveFlags, globalFlags...)
	serveFlags = append(serveFlags, serverFlags...)

	app := &cli.App{
		Name:   "shortcut-signing-server",
		Usage:  "A simple server for signing iOS/macOS shortcuts over http",
		Before: altsrc.InitInputSourceWithContext(globalFlags, altsrc.NewYamlSourceFromFlagFunc("config")),
		Commands: []*cli.Command{
			{
				Name:  "serve",
				Usage: "Start the server",
				Action: func(ctx *cli.Context) error {
					return runApp(ctx)
				},
				Flags:  serveFlags,
				Before: altsrc.InitInputSourceWithContext(serveFlags, altsrc.NewYamlSourceFromFlagFunc("config")),
			},
		},

		Flags: globalFlags,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func runApp(ctx *cli.Context) error {

	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	setupLogging(ctx)

	var options []internal.ServerOption

	processFlags(ctx, &options)

	server := internal.NewServer(ctx.String("listen"), options...)

	logger.Debug().Interface("server", server).Msg("inspecting server")

	return server.Listen(ctx.Args().First())
}

func processFlags(ctx *cli.Context, options *[]internal.ServerOption) {

	if ctx.Bool("tls") {
		*options = append(*options, internal.EnableTls(ctx.String("tls-cert"), ctx.String("tls-key")))
	} else if ctx.Path("tls-cert") != "" || ctx.Path("tls-key") != "" {
		logger.Warn().Msg("Tls is disabled, but tls-cert and/or tls-key are set")
	}

	if ctx.Bool("re") {
		*options = append(*options, internal.EnableFullErrorsResponse())
	}

	if templates := ctx.Path("templates"); templates != "" {
		*options = append(*options, internal.Templates(templates))
	}

}

func setupLogging(ctx *cli.Context) {
	logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

	if ctx.Bool("debug") && ctx.String("log-level") == "" {
		logger = logger.Level(zerolog.DebugLevel)
	}

	if ctx.String("log-level") != "" {
		switch ctx.String("log-level") {
		case "debug":
			logger = logger.Level(zerolog.DebugLevel)
		case "info":
			logger = logger.Level(zerolog.InfoLevel)
		case "warn":
			logger = logger.Level(zerolog.WarnLevel)
		case "error":
			logger = logger.Level(zerolog.ErrorLevel)
		case "fatal":
			logger = logger.Level(zerolog.FatalLevel)
		case "panic":
			logger = logger.Level(zerolog.PanicLevel)
		case "trace":
			logger = logger.Level(zerolog.TraceLevel)
		default:
			logger = logger.Level(zerolog.InfoLevel)
		}
	}

	var output io.Writer

	if ctx.String("log-format") != "" {
		switch ctx.String("log-format") {
		case "text":
			output = zerolog.ConsoleWriter{Out: os.Stdout}
		case "json":
			output = os.Stdout
		}

	}

	if ctx.Bool("quiet") {
		output = io.Discard
	}

	logger = logger.Output(output)

	if ctx.String("log-file") != "" {
		var err error
		logFile, err = os.OpenFile(ctx.String("log-file"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logger.Panic().Err(err).Msg("Opening log file")
		}

		output = io.MultiWriter(output, logFile)
	}

	logger = logger.Output(output)

	internal.SetLogger(logger)
}
