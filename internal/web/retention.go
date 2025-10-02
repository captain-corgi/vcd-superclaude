package web

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/storage"
)

// RetentionJob handles cleanup of old audit events
type RetentionJob struct {
	storage       *storage.Storage
	retentionDays int
	logger        *zap.Logger
	stopChan      chan struct{}
}

// NewRetentionJob creates a new retention job
func NewRetentionJob(storage *storage.Storage, retentionDays int, logger *zap.Logger) *RetentionJob {
	return &RetentionJob{
		storage:       storage,
		retentionDays: retentionDays,
		logger:        logger,
		stopChan:      make(chan struct{}),
	}
}

// Start begins the retention job with daily cleanup
func (r *RetentionJob) Start(ctx context.Context) {
	if r.retentionDays <= 0 {
		r.logger.Info("Audit retention disabled (retention_days <= 0)")
		return
	}

	r.logger.Info("Starting audit retention job",
		zap.Int("retention_days", r.retentionDays))

	// Run initial cleanup
	r.runCleanup(ctx)

	// Schedule daily cleanup
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.runCleanup(ctx)
		case <-r.stopChan:
			r.logger.Info("Audit retention job stopped")
			return
		case <-ctx.Done():
			r.logger.Info("Audit retention job stopped due to context cancellation")
			return
		}
	}
}

// Stop stops the retention job
func (r *RetentionJob) Stop() {
	close(r.stopChan)
}

// runCleanup performs the actual cleanup of old audit events
func (r *RetentionJob) runCleanup(ctx context.Context) {
	start := time.Now()

	rowsAffected, err := r.storage.CleanupAuditEvents(ctx, r.retentionDays)
	if err != nil {
		r.logger.Error("Failed to cleanup audit events",
			zap.Error(err),
			zap.Int("retention_days", r.retentionDays))
		return
	}

	duration := time.Since(start)
	r.logger.Info("Audit retention cleanup completed",
		zap.Int64("rows_deleted", rowsAffected),
		zap.Int("retention_days", r.retentionDays),
		zap.Duration("duration", duration))
}
