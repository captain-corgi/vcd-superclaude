package audit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/captain-corgi/multi-agent-system/pkg/metrics"
)

// Recorder implements the AuditRecorder interface with async buffered writes
type Recorder struct {
	storage  Storage
	redactor *Redactor
	logger   *zap.Logger
	events   chan AuditEvent
	done     chan struct{}
	wg       sync.WaitGroup
	metrics  *Metrics
}

// Storage interface for audit event persistence
type Storage interface {
	InsertAuditEvent(ctx context.Context, event AuditEvent) error
}

// Metrics tracks audit recorder performance
type Metrics struct {
	Queued  int64
	Written int64
	Dropped int64
	Latency time.Duration
	mu      sync.RWMutex
}

// NewRecorder creates a new audit recorder
func NewRecorder(storage Storage, redactor *Redactor, logger *zap.Logger, bufferSize int) *Recorder {
	return &Recorder{
		storage:  storage,
		redactor: redactor,
		logger:   logger,
		events:   make(chan AuditEvent, bufferSize),
		done:     make(chan struct{}),
		metrics:  &Metrics{},
	}
}

// Start begins the async processing of audit events
func (r *Recorder) Start(ctx context.Context) {
	r.wg.Add(1)
	go r.processEvents(ctx)
}

// Record queues an audit event for async processing
func (r *Recorder) Record(event AuditEvent) error {
	select {
	case r.events <- event:
		r.metrics.incrementQueued()
		metrics.IncrementAuditQueued()
		return nil
	default:
		r.metrics.incrementDropped()
		metrics.IncrementAuditDropped()
		r.logger.Warn("Audit event dropped due to full buffer",
			zap.String("event_id", event.EventID),
			zap.String("category", event.Category))
		return fmt.Errorf("audit buffer full, event dropped")
	}
}

// processEvents processes audit events from the buffer
func (r *Recorder) processEvents(ctx context.Context) {
	defer r.wg.Done()

	for {
		select {
		case event := <-r.events:
			r.processEvent(ctx, event)
		case <-r.done:
			// Process remaining events before shutdown
			for {
				select {
				case event := <-r.events:
					r.processEvent(ctx, event)
				default:
					return
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

// processEvent processes a single audit event
func (r *Recorder) processEvent(ctx context.Context, event AuditEvent) {
	start := time.Now()

	// Redact sensitive data if payload exists
	if event.Payload != nil {
		redacted, hasRedacted := r.redactor.RedactPayload(event.Payload)
		event.Payload = redacted
		event.Redacted = hasRedacted
	}

	// Insert into storage
	if err := r.storage.InsertAuditEvent(ctx, event); err != nil {
		r.logger.Error("Failed to insert audit event",
			zap.String("event_id", event.EventID),
			zap.Error(err))
		r.metrics.incrementDropped()
		return
	}

	// Update metrics
	r.metrics.incrementWritten()
	r.metrics.updateLatency(time.Since(start))
	metrics.IncrementAuditWritten()
	metrics.UpdateAuditLatency(time.Since(start))

	r.logger.Debug("Audit event recorded",
		zap.String("event_id", event.EventID),
		zap.String("category", event.Category),
		zap.String("action", event.Action))
}

// Close gracefully shuts down the recorder
func (r *Recorder) Close() error {
	close(r.done)
	r.wg.Wait()
	close(r.events)
	return nil
}

// GetMetrics returns current recorder metrics
func (r *Recorder) GetMetrics() Metrics {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()
	return *r.metrics
}

// incrementQueued increments the queued counter
func (m *Metrics) incrementQueued() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Queued++
}

// incrementWritten increments the written counter
func (m *Metrics) incrementWritten() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Written++
}

// incrementDropped increments the dropped counter
func (m *Metrics) incrementDropped() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Dropped++
}

// updateLatency updates the latency metric
func (m *Metrics) updateLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Latency = latency
}
