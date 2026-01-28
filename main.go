package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	watcher := NewWatcher(cfg)
	watcher.Run(ctx)

	slog.Info("shutdown complete")
}
