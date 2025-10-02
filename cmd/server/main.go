package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/captain-corgi/multi-agent-system/internal/web"
	"github.com/captain-corgi/multi-agent-system/pkg/config"
	"github.com/captain-corgi/multi-agent-system/pkg/logger"
	"github.com/captain-corgi/multi-agent-system/pkg/storage"
	"go.uber.org/zap"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/app.yaml")
	if err != nil {
		// Try loading from environment if config file not found
		cfg, err = config.LoadFromEnv()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		log.Println("Using environment configuration")
	}

	// Initialize logger from config
	var appLogger *logger.Logger
	if cfg.Logging.Datadog.Enabled {
		appLogger = logger.NewDatadogLogger(cfg.Logging, cfg.App)
	} else {
		appLogger = logger.NewFromConfig(
			cfg.Logging.Level,
			cfg.Logging.Format,
			cfg.Logging.Output,
			cfg.Logging.File,
			cfg.Logging.Development,
		)
	}
	defer appLogger.Sync()

	// Start Datadog tracer if enabled
	if cfg.Logging.Datadog.Enabled && cfg.Logging.Datadog.InjectAPM {
		// Set environment variables for tracer
		if cfg.Logging.Datadog.Service != "" {
			os.Setenv("DD_SERVICE", cfg.Logging.Datadog.Service)
		}
		if cfg.Logging.Datadog.Env != "" {
			os.Setenv("DD_ENV", cfg.Logging.Datadog.Env)
		}
		if cfg.Logging.Datadog.Version != "" {
			os.Setenv("DD_VERSION", cfg.Logging.Datadog.Version)
		}

		// Start tracer
		tracer.Start(
			tracer.WithService(cfg.App.Name),
			tracer.WithEnv(cfg.Logging.Datadog.Env),
		)
		defer tracer.Stop()

		appLogger.Info("Datadog tracer started",
			zap.String("service", cfg.App.Name),
			zap.String("env", cfg.Logging.Datadog.Env),
			zap.String("version", cfg.Logging.Datadog.Version))
	}

	// Initialize storage
	storage, err := storage.NewStorage(
		cfg.Database.URL,
		cfg.Database.RedisURL,
		appLogger.Zap(),
	)
	if err != nil {
		appLogger.Zap().Fatal("Failed to initialize storage",
			zap.Error(err))
	}
	defer storage.Close()

	// Initialize web server
	server := web.NewServer(cfg, storage, appLogger.Zap())

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		appLogger.Info("Starting server",
			zap.String("app", cfg.App.Name),
			zap.String("version", cfg.App.Version),
			zap.Int("port", cfg.App.Port),
			zap.Bool("debug", cfg.App.Debug))

		addr := fmt.Sprintf(":%d", cfg.App.Port)
		err := server.Start(addr)
		if err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal
	select {
	case sig := <-shutdown:
		appLogger.Info("Shutting down server...",
			zap.String("signal", sig.String()))
	case err := <-serverErrors:
		appLogger.Error("Server error",
			zap.Error(err))
		return
	}

	// Graceful shutdown timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	appLogger.Info("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown",
			zap.Error(err))
	}

	appLogger.Info("Server exited gracefully")
}
