package worker

import (
	"context"
	"time"

	"github.com/devgugga/galaxy-pay/internal/infrastructure/cache"
	"github.com/devgugga/galaxy-pay/pkg/logger"
)

// IdempotencyCleanupWorker handles periodic cleanup of expired idempotency keys
type IdempotencyCleanupWorker struct {
	store    *cache.IdempotencyStore
	interval time.Duration
}

// NewIdempotencyCleanupWorker creates a new cleanup worker
func NewIdempotencyCleanupWorker(store *cache.IdempotencyStore, interval time.Duration) *IdempotencyCleanupWorker {
	if interval == 0 {
		interval = 1 * time.Hour
	}
	return &IdempotencyCleanupWorker{
		store:    store,
		interval: interval,
	}
}

// Start starts the cleanup worker in a background goroutine
func (w *IdempotencyCleanupWorker) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		// Run immediately on start
		w.cleanupExpiredKeys(ctx)

		for {
			select {
			case <-ctx.Done():
				logger.Log.Info("Idempotency cleanup worker stopped")
				return
			case <-ticker.C:
				w.cleanupExpiredKeys(ctx)
			}
		}
	}()

	logger.Log.Info("Idempotency cleanup worker started",
		logger.String("interval", w.interval.String()),
	)
}

// cleanupExpiredKeys removes expired keys from the database
// Note: Redis keys expire automatically via TTL, so we only clean the database
func (w *IdempotencyCleanupWorker) cleanupExpiredKeys(ctx context.Context) {
	deleted, err := w.store.CleanupExpiredKeys(ctx)
	if err != nil {
		logger.Log.Error("Failed to cleanup expired idempotency keys",
			logger.Error(err),
		)
		return
	}

	if deleted > 0 {
		logger.Log.Info("Cleaned up expired idempotency keys",
			logger.Int("count", deleted),
		)
	}
}

