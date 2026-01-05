package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
type IdempotencyStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewIdempotencyStore creates a new idempotency store
func NewIdempotencyStore(client *redis.Client, ttl time.Duration) *IdempotencyStore {
	if ttl == 0 {
		ttl = DefaultIdempotencyTTL
	}
	return &IdempotencyStore{
		client: client,
		ttl:    ttl,
	}
}

// Get retrieves a cached response for an idempotency key
func (s *IdempotencyStore) Get(ctx context.Context, key string) (*IdempotencyResponse, error) {
	if s.client == nil {
		return nil, fmt.Errorf("Redis client is not initialized")
	}

	fullKey := IdempotencyKeyPrefix + key
	data, err := s.client.Get(ctx, fullKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Key doesn't exist
		}
		return nil, fmt.Errorf("failed to get idempotency key: %w", err)
	}

	var response IdempotencyResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal idempotency response: %w", err)
	}

	return &response, nil
}

// Set stores a response for an idempotency key
func (s *IdempotencyStore) Set(ctx context.Context, key string, response *IdempotencyResponse) error {
	if s.client == nil {
		return fmt.Errorf("Redis client is not initialized")
	}

	fullKey := IdempotencyKeyPrefix + key
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal idempotency response: %w", err)
	}

	if err := s.client.Set(ctx, fullKey, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set idempotency key: %w", err)
	}

	return nil
}

// Exists checks if an idempotency key exists
func (s *IdempotencyStore) Exists(ctx context.Context, key string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("Redis client is not initialized")
	}

	fullKey := IdempotencyKeyPrefix + key
	count, err := s.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check idempotency key existence: %w", err)
	}

	return count > 0, nil
}

// Delete removes an idempotency key
func (s *IdempotencyStore) Delete(ctx context.Context, key string) error {
	if s.client == nil {
		return fmt.Errorf("Redis client is not initialized")
	}

	fullKey := IdempotencyKeyPrefix + key
	if err := s.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete idempotency key: %w", err)
	}

	return nil
}

