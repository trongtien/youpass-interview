package datastore

import (
	"context"
	"crypto/rand"
	"fmt"
	"service-share-link/internal/share-link/model"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	viewsPendingKey       = keyPrefix + "{views}:pending"
	viewsInflightPrefix   = keyPrefix + "{views}:inflight:"
	viewsInflightIndexKey = keyPrefix + "{views}:inflight_index"
)

type ViewBatch struct {
	Key    string
	Deltas map[model.Code]int64
}

type ViewCounter interface {
	Incr(ctx context.Context, code model.Code) error
	Claim(ctx context.Context) (*ViewBatch, error)
	ClaimStale(ctx context.Context, olderThan time.Duration, limit int64) ([]*ViewBatch, error)
	Ack(ctx context.Context, key string) (acked bool, err error)
	Release(ctx context.Context, key string) error
}

type RedisViewCounter struct {
	client redis.Cmdable
}

func NewRedisViewCounter(client redis.Cmdable) *RedisViewCounter {
	return &RedisViewCounter{
		client: client,
	}
}

func (c *RedisViewCounter) Incr(ctx context.Context, code model.Code) error {
	if err := c.client.HIncrBy(ctx, viewsPendingKey, string(code), 1).Err(); err != nil {
		return fmt.Errorf("[ViewCounter] Incr view %w", err)
	}

	return nil
}

func (c *RedisViewCounter) Claim(ctx context.Context) (*ViewBatch, error) {
	return c.claim(ctx, viewsPendingKey, false)
}

func (c *RedisViewCounter) ClaimStale(ctx context.Context, olderThan time.Duration, limit int64) ([]*ViewBatch, error) {
	keys, err := c.client.ZRangeByScore(ctx, viewsInflightIndexKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   strconv.FormatInt(time.Now().Add(-olderThan).UnixMilli(), 10),
		Count: limit,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("[ViewCounter] List stale batches %w", err)
	}

	var batches []*ViewBatch
	for _, key := range keys {
		batch, err := c.claim(ctx, key, true)
		if err != nil {
			return nil, err
		}
		if batch != nil {
			batches = append(batches, batch)
		}
	}

	return batches, nil
}

func (c *RedisViewCounter) claim(ctx context.Context, src string, srcIndexed bool) (*ViewBatch, error) {
	key := viewsInflightPrefix + rand.Text()
	_, err := c.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		p.Rename(ctx, src, key)
		p.ZAdd(ctx, viewsInflightIndexKey, redis.Z{Score: float64(time.Now().UnixMilli()), Member: key})
		if srcIndexed {
			p.ZRem(ctx, viewsInflightIndexKey, src)
		}
		return nil
	})
	if isNoSuchKey(err) {
		_ = c.client.ZRem(ctx, viewsInflightIndexKey, key).Err()
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("[ViewCounter] Claim batch %w", err)
	}

	raw, err := c.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("[ViewCounter] Read batch %w", err)
	}

	deltas := make(map[model.Code]int64, len(raw))
	for code, value := range raw {
		delta, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("[ViewCounter] Parse delta code=%s %w", code, err)
		}
		deltas[model.Code(code)] = delta
	}

	return &ViewBatch{Key: key, Deltas: deltas}, nil
}

func (c *RedisViewCounter) Ack(ctx context.Context, key string) (bool, error) {
	var deleted *redis.IntCmd
	_, err := c.client.TxPipelined(ctx, func(p redis.Pipeliner) error {
		deleted = p.Del(ctx, key)
		p.ZRem(ctx, viewsInflightIndexKey, key)
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("[ViewCounter] Ack batch %w", err)
	}

	return deleted.Val() == 1, nil
}

func (c *RedisViewCounter) Release(ctx context.Context, key string) error {
	if err := c.client.ZAddXX(ctx, viewsInflightIndexKey, redis.Z{Score: 0, Member: key}).Err(); err != nil {
		return fmt.Errorf("[ViewCounter] Release batch %w", err)
	}

	return nil
}

func isNoSuchKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such key")
}
