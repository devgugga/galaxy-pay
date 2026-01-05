package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/devgugga/galaxy-pay/ent"
	"github.com/devgugga/galaxy-pay/ent/idempotencykey"
	"github.com/devgugga/galaxy-pay/pkg/logger"
	"github.com/redis/go-redis/v9"
)

const (
	// IdempotencyKeyPrefix is the prefix for idempotency keys in Redis
	IdempotencyKeyPrefix = "idempotency:"
	// DefaultIdempotencyTTL is the default TTL for idempotency keys (24 hours)
	DefaultIdempotencyTTL = 24 * time.Hour
)

// IdempotencyResponse represents a cached response
type IdempotencyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// IdempotencyStore handles idempotency key storage and retrieval
// Implements Cache-Aside pattern: Redis (cache) + PostgreSQL (persistence)
type IdempotencyStore struct {
	redisClient *redis.Client
	entClient   *ent.Client
	ttl         time.Duration
	requestID   string // For logging context
}

// NewIdempotencyStore creates a new idempotency store with Redis and Ent clients
func NewIdempotencyStore(redisClient *redis.Client, entClient *ent.Client, ttl time.Duration) *IdempotencyStore {
	if ttl == 0 {
		ttl = DefaultIdempotencyTTL
	}
	return &IdempotencyStore{
		redisClient: redisClient,
		entClient:   entClient,
		ttl:         ttl,
	}
}

// WithRequestID sets the request ID for logging context
func (s *IdempotencyStore) WithRequestID(requestID string) *IdempotencyStore {
	s.requestID = requestID
	return s
}

// Get retrieves a cached response for an idempotency key (Cache-Aside pattern)
// 1. Tries Redis first (fast cache)
// 2. If not found, tries database (persistence)
// 3. If found in DB, repopulates Redis async
func (s *IdempotencyStore) Get(ctx context.Context, key string) (*IdempotencyResponse, error) {
	// 1. Try Redis first (cache)
	cached, err := s.getFromRedis(ctx, key)
	if err == nil && cached != nil {
		logger.WithRequestID(s.requestID).Debug("Idempotency key found in Redis cache",
			logger.String("key", key),
		)
		return cached, nil
	}

	// Log Redis miss (non-critical, fail-open)
	if err != nil && err != redis.Nil {
		logger.WithRequestID(s.requestID).Warn("Redis lookup failed, falling back to database",
			logger.String("key", key),
			logger.Error(err),
		)
	}

	// 2. Try database (persistence)
	dbResult, err := s.getFromDB(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get idempotency key from database: %w", err)
	}

	if dbResult != nil {
		// 3. Repopulate Redis async (non-blocking)
		go func() {
			repopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.setInRedis(repopCtx, key, dbResult); err != nil {
				logger.WithRequestID(s.requestID).Warn("Failed to repopulate Redis cache",
					logger.String("key", key),
					logger.Error(err),
				)
			} else {
				logger.WithRequestID(s.requestID).Debug("Repopulated Redis cache from database",
					logger.String("key", key),
				)
			}
		}()

		logger.WithRequestID(s.requestID).Info("Idempotency key found in database",
			logger.String("key", key),
		)
		return dbResult, nil
	}

	// Not found in either store
	return nil, nil
}

// Set stores a response for an idempotency key (Write-Through pattern)
// 1. Saves to database first (critical, must persist)
// 2. Then saves to Redis (cache, can fail silently)
func (s *IdempotencyStore) Set(ctx context.Context, key string, response *IdempotencyResponse) error {
	// 1. Save to database first (critical)
	if err := s.setInDB(ctx, key, response); err != nil {
		logger.WithRequestID(s.requestID).Error("Failed to save idempotency key to database",
			logger.String("key", key),
			logger.Error(err),
		)
		// Try Redis as last resort fallback
		if redisErr := s.setInRedis(ctx, key, response); redisErr != nil {
			return fmt.Errorf("failed to save to both database and Redis: db=%w, redis=%w", err, redisErr)
		}
		logger.WithRequestID(s.requestID).Warn("Saved to Redis as fallback (database failed)",
			logger.String("key", key),
		)
		return err // Return DB error even if Redis succeeded
	}

	// 2. Save to Redis async (non-blocking, can fail silently)
	go func() {
		redisCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.setInRedis(redisCtx, key, response); err != nil {
			logger.WithRequestID(s.requestID).Warn("Failed to save idempotency key to Redis cache",
				logger.String("key", key),
				logger.Error(err),
			)
		} else {
			logger.WithRequestID(s.requestID).Debug("Saved idempotency key to Redis cache",
				logger.String("key", key),
			)
		}
	}()

	logger.WithRequestID(s.requestID).Info("Saved idempotency key",
		logger.String("key", key),
		logger.Int("status_code", response.StatusCode),
	)

	return nil
}

// Exists checks if an idempotency key exists (checks both Redis and DB)
func (s *IdempotencyStore) Exists(ctx context.Context, key string) (bool, error) {
	// Check Redis first
	if s.redisClient != nil {
		fullKey := IdempotencyKeyPrefix + key
		count, err := s.redisClient.Exists(ctx, fullKey).Result()
		if err == nil && count > 0 {
			return true, nil
		}
	}

	// Check database
	if s.entClient != nil {
		exists, err := s.entClient.IdempotencyKey.
			Query().
			Where(idempotencykey.Key(key)).
			Exist(ctx)
		if err != nil {
			return false, fmt.Errorf("failed to check idempotency key existence in database: %w", err)
		}
		return exists, nil
	}

	return false, nil
}

// Delete removes an idempotency key from both Redis and database
func (s *IdempotencyStore) Delete(ctx context.Context, key string) error {
	var wg sync.WaitGroup
	var redisErr, dbErr error

	// Delete from Redis
	if s.redisClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fullKey := IdempotencyKeyPrefix + key
			redisErr = s.redisClient.Del(ctx, fullKey).Err()
		}()
	}

	// Delete from database
	if s.entClient != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, dbErr = s.entClient.IdempotencyKey.
				Delete().
				Where(idempotencykey.Key(key)).
				Exec(ctx)
		}()
	}

	wg.Wait()

	if dbErr != nil {
		return fmt.Errorf("failed to delete idempotency key from database: %w", dbErr)
	}
	if redisErr != nil {
		logger.WithRequestID(s.requestID).Warn("Failed to delete idempotency key from Redis",
			logger.String("key", key),
			logger.Error(redisErr),
		)
	}

	return nil
}

// Private methods

// getFromRedis retrieves from Redis cache
func (s *IdempotencyStore) getFromRedis(ctx context.Context, key string) (*IdempotencyResponse, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("Redis client is not initialized")
	}

	fullKey := IdempotencyKeyPrefix + key
	data, err := s.redisClient.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Key doesn't exist
		}
		return nil, fmt.Errorf("failed to get from Redis: %w", err)
	}

	var response IdempotencyResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Redis response: %w", err)
	}

	return &response, nil
}

// getFromDB retrieves from PostgreSQL database
func (s *IdempotencyStore) getFromDB(ctx context.Context, key string) (*IdempotencyResponse, error) {
	if s.entClient == nil {
		return nil, fmt.Errorf("Ent client is not initialized")
	}

	ik, err := s.entClient.IdempotencyKey.
		Query().
		Where(
			idempotencykey.Key(key),
			idempotencykey.ExpiresAtGT(time.Now()), // Only non-expired keys
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // Key doesn't exist
		}
		return nil, fmt.Errorf("failed to query database: %w", err)
	}

	return &IdempotencyResponse{
		StatusCode: ik.StatusCode,
		Headers:    ik.ResponseHeaders,
		Body:       ik.ResponseBody,
	}, nil
}

// setInRedis saves to Redis cache
func (s *IdempotencyStore) setInRedis(ctx context.Context, key string, response *IdempotencyResponse) error {
	if s.redisClient == nil {
		return fmt.Errorf("Redis client is not initialized")
	}

	fullKey := IdempotencyKeyPrefix + key
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if err := s.redisClient.Set(ctx, fullKey, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set in Redis: %w", err)
	}

	return nil
}

// setInDB saves to PostgreSQL database
func (s *IdempotencyStore) setInDB(ctx context.Context, key string, response *IdempotencyResponse) error {
	if s.entClient == nil {
		return fmt.Errorf("Ent client is not initialized")
	}

	expiresAt := time.Now().Add(s.ttl)

	// Check if key already exists
	existing, err := s.entClient.IdempotencyKey.
		Query().
		Where(idempotencykey.Key(key)).
		Only(ctx)

	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("failed to check existing key: %w", err)
	}

	if existing != nil {
		// Update existing record
		_, err = s.entClient.IdempotencyKey.
			UpdateOneID(existing.ID).
			SetStatusCode(response.StatusCode).
			SetResponseBody(response.Body).
			SetResponseHeaders(response.Headers).
			SetExpiresAt(expiresAt).
			Save(ctx)
	} else {
		// Create new record
		_, err = s.entClient.IdempotencyKey.
			Create().
			SetKey(key).
			SetStatusCode(response.StatusCode).
			SetResponseBody(response.Body).
			SetResponseHeaders(response.Headers).
			SetExpiresAt(expiresAt).
			Save(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to save to database: %w", err)
	}

	return nil
}

// CleanupExpiredKeys removes expired keys from the database
// Returns the number of deleted keys
// Note: Redis keys expire automatically via TTL, so this only cleans the database
func (s *IdempotencyStore) CleanupExpiredKeys(ctx context.Context) (int, error) {
	if s.entClient == nil {
		return 0, nil // No database, nothing to clean
	}

	now := time.Now()
	deleted, err := s.entClient.IdempotencyKey.
		Delete().
		Where(idempotencykey.ExpiresAtLT(now)).
		Exec(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired keys: %w", err)
	}

	return deleted, nil
}
