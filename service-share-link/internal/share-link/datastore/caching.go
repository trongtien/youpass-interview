package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"service-share-link/internal/share-link/model"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix      = "share_link:"
	notFoundMarker = "__nf__"

	ttlActive   = 1 * time.Hour
	ttlDisabled = 10 * time.Minute
	ttlDeleted  = 24 * time.Hour
	ttlNotFound = 60 * time.Second

	ttlJitterRatio = 0.1
)

type Cache interface {
	Get(ctx context.Context, code model.Code) (link *model.ShareLink, hit bool, err error)
	Set(ctx context.Context, link *model.ShareLink) error
	SetIfAbsent(ctx context.Context, link *model.ShareLink) error
	SetNotFound(ctx context.Context, code model.Code) error
	Delete(ctx context.Context, code model.Code) error
}

var _ Cache = (*RedisCache)(nil)

type RedisCache struct {
	client redis.Cmdable
}

func NewRedisCache(client redis.Cmdable) *RedisCache {
	return &RedisCache{
		client: client,
	}
}

func (c *RedisCache) Get(ctx context.Context, code model.Code) (*model.ShareLink, bool, error) {
	data, err := c.client.Get(ctx, cacheKey(code)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("[Cache] Get share link %w", err)
	}

	if string(data) == notFoundMarker {
		return nil, true, nil
	}

	var cached model.ShareLink
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, false, fmt.Errorf("[Cache] Unmarshal share link %w", err)
	}

	return &cached, true, nil
}

func (c *RedisCache) Set(ctx context.Context, link *model.ShareLink) error {
	data, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("[Cache] Marshal share link %w", err)
	}

	if err := c.client.Set(ctx, cacheKey(link.Code), data, withJitter(ttlByStatus(link.Status))).Err(); err != nil {
		return fmt.Errorf("[Cache] Set share link %w", err)
	}

	return nil
}

func (c *RedisCache) SetIfAbsent(ctx context.Context, link *model.ShareLink) error {
	data, err := json.Marshal(link)
	if err != nil {
		return fmt.Errorf("[Cache] Marshal share link %w", err)
	}

	if err := c.client.SetNX(ctx, cacheKey(link.Code), data, withJitter(ttlByStatus(link.Status))).Err(); err != nil {
		return fmt.Errorf("[Cache] Set share link if absent %w", err)
	}

	return nil
}

func (c *RedisCache) SetNotFound(ctx context.Context, code model.Code) error {
	if err := c.client.Set(ctx, cacheKey(code), notFoundMarker, withJitter(ttlNotFound)).Err(); err != nil {
		return fmt.Errorf("[Cache] Set not found marker %w", err)
	}

	return nil
}

func (c *RedisCache) Delete(ctx context.Context, code model.Code) error {
	if err := c.client.Del(ctx, cacheKey(code)).Err(); err != nil {
		return fmt.Errorf("[Cache] Delete share link %w", err)
	}

	return nil
}

func cacheKey(code model.Code) string {
	return keyPrefix + string(code)
}

func ttlByStatus(status model.Status) time.Duration {
	switch status {
	case model.StatusDisabled:
		return ttlDisabled
	case model.StatusDelete:
		return ttlDeleted
	default:
		return ttlActive
	}
}

func withJitter(ttl time.Duration) time.Duration {
	spread := int64(float64(ttl) * ttlJitterRatio)
	if spread <= 0 {
		return ttl
	}

	return ttl + time.Duration(rand.Int64N(2*spread+1)-spread)
}
