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

	"github.com/anh-tuan-l/multi-agent-system/internal/web"
	"github.com/anh-tuan-l/multi-agent-system/pkg/config"
	"github.com/anh-tuan-l/multi-agent-system/pkg/logger"
	"github.com/anh-tuan-l/multi-agent-system/pkg/storage"
	"go.uber.org/zap"
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

	// Initialize logger
	logConfig := zap.NewProductionConfig()
	if cfg.App.Debug {
		logConfig.Development = true
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger := logger.NewWithConfig(logConfig)
	defer logger.Sync()

	// Initialize storage
	storage, err := storage.NewStorage(
		cfg.Database.URL,
		cfg.Database.RedisURL,
		logger.Zap(),
	)
	if err != nil {
		logger.Zap().Fatal("Failed to initialize storage",
			zap.Error(err))
	}
	defer storage.Close()

	// Initialize web server
	server := web.NewServer(cfg, storage, logger.Zap())

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Starting server",
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
		logger.Info("Shutting down server...",
			zap.String("signal", sig.String()))
	case err := <-serverErrors:
		logger.Error("Server error",
			zap.Error(err))
		return
	}

	// Graceful shutdown timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown",
			zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}
