package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "/etc/ipmi-cert-pusher/config.yaml", "path to config file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("config loaded",
		"servers", len(cfg.Servers),
		"poll_interval", cfg.PollInterval,
		"saa_binary", cfg.SAABinary,
	)

	for _, s := range cfg.Servers {
		slog.Info("server configured", "name", s.Name, "host", s.IPMIHost)
	}

	go func() {
		slog.Info("starting metrics server", "addr", ":8080")
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":8080", mux); err != nil {
			slog.Error("metrics server failed", "error", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	watcher := NewWatcher(cfg)
	watcher.Run(ctx)

	slog.Info("shutdown complete")
}
