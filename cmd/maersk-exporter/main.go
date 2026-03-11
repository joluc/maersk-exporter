package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joluc/maersk-exporter/pkg/config"
	"github.com/joluc/maersk-exporter/pkg/exporter"
)

func main() {
	// Configure slog
	logger := slog.Default()
	slog.SetDefault(logger)

	cfg, err := config.ParseFlags()
	if err != nil {
		slog.Error("failed to parse flags", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if cfg.MaerskConsumerKey == "" {
		slog.Error("maersk-consumer-key is required (use flag or MAERSK_CONSUMER_KEY env)")
		os.Exit(1)
	}
	if cfg.AISStreamAPIKey == "" {
		slog.Error("aisstream-api-key is required (use flag or AISSTREAM_API_KEY env)")
		os.Exit(1)
	}
	if len(cfg.FilterVesselNames) == 0 {
		slog.Error("vessel-name-filter is required (e.g., 'MAERSK')")
		os.Exit(1)
	}
	if cfg.VesselsRefreshInterval <= 0 {
		slog.Error("vessels-refresh-interval must be > 0")
		os.Exit(1)
	}
	if cfg.RequestTimeout <= 0 {
		slog.Error("request-timeout must be > 0")
		os.Exit(1)
	}

	exp := exporter.New(cfg)

	// Setup context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start background processes (WebSocket + periodic refresh)
	exp.Start(ctx)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc(cfg.MetricsPath, exp.MetricsHandler)
	mux.HandleFunc("/healthz", exp.HealthHandler)
	mux.HandleFunc("/", exp.RootHandler())

	srv := &http.Server{
		Addr:              cfg.ListenAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server in goroutine
	go func() {
		slog.Info("starting maersk fleet & position monitoring exporter",
			"address", cfg.ListenAddress,
			"metrics_path", cfg.MetricsPath,
			"vessels_refresh_interval", cfg.VesselsRefreshInterval,
			"vessel_filters", cfg.FilterVesselNames)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	slog.Info("shutting down gracefully")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("shutdown complete")
}
