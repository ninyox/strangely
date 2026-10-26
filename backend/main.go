package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strgback/config"
	"strgback/handlers"
	"strgback/routes"
	"strgback/store"
)

// janitorInterval is how often expired sessions, stale queue entries and
// abandoned matches are swept. The hot paths also clean up lazily; this is the
// backstop for a process nobody is using.
const janitorInterval = 15 * time.Second

func main() {
	config.LoadEnv()

	logger := newLogger(config.App)
	slog.SetDefault(logger)

	st := store.New(store.Options{
		SessionTTL:           config.App.SessionTTL,
		QueueTTL:             config.App.QueueTTL,
		MatchIdleTTL:         config.App.MatchIdleTTL,
		MaxMatchDuration:     config.App.MaxMatchDuration,
		SkipLock:             config.App.SkipLock,
		ReportsPerTenMinutes: config.App.RateLimit.ReportPerTenMinutes,
	}, logger)

	st.StartJanitor(janitorInterval)
	defer st.StopJanitor()

	api := handlers.New(st, logger)
	router := routes.NewRouter(api)

	server := &http.Server{
		Addr:              config.App.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 14,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	go func() {
		logger.Info("strangerly api listening",
			"addr", config.App.ListenAddr,
			"environment", config.App.Environment,
			"allowed_origins", config.App.AllowedOrigins,
			"skip_lock", config.App.SkipLock.String(),
			"turn_configured", len(config.App.ICE.TURNURLs) > 0,
			"turn_urls", len(config.App.ICE.TURNURLs),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	logger.Info("shutting down")
	api.MarkNotReady()
	api.Hub.Close()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}

// newLogger uses structured JSON in production and human readable text
// locally. Session tokens are never logged anywhere in this service.
func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.SlogLevel()}

	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, opts)
	if !cfg.IsProduction() {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
