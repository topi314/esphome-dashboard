package main

import (
	"embed"
	"flag"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/charmbracelet/log"
	"github.com/muesli/termenv"

	"github.com/topi314/esphome-dashboard/dashboard"
)

//go:embed templates/*.gohtml
var templates embed.FS

func main() {
	cfgPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	cfg, err := dashboard.LoadConfig(*cfgPath)
	if err != nil {
		slog.Error("Error while loading config", slog.Any("err", err))
		return
	}

	setupLogger(cfg.Log)

	version := "unknown"
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	slog.Info("Starting dashboard...", slog.String("version", version), slog.String("go_version", runtime.Version()))
	slog.Info("Config loaded", slog.Any("config", cfg))

	var t fs.FS
	if cfg.Dev {
		t = os.DirFS(".")
	} else {
		t = templates
	}

	s := dashboard.New(cfg, t)
	go s.Start()

	slog.Info("Dashboard started", slog.Any("addr", cfg.ListenAddr))
	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM)
	<-si
}

func setupLogger(cfg dashboard.LogConfig) {
	var formatter log.Formatter
	switch cfg.Format {
	case dashboard.LogFormatJSON:
		formatter = log.JSONFormatter
	case dashboard.LogFormatText:
		formatter = log.TextFormatter
	case dashboard.LogFormatLogFMT:
		formatter = log.LogfmtFormatter
	default:
		slog.Error("Unknown log format", slog.String("format", string(cfg.Format)))
		os.Exit(-1)
	}

	handler := log.NewWithOptions(os.Stdout, log.Options{
		Level:           log.Level(cfg.Level),
		ReportTimestamp: true,
		ReportCaller:    cfg.AddSource,
		Formatter:       formatter,
	})
	if cfg.Format == dashboard.LogFormatText && !cfg.NoColor {
		handler.SetColorProfile(termenv.TrueColor)
	}

	slog.SetDefault(slog.New(handler))
}
