package redis

import (
	"context"
	"fmt"
	"service-share-link/internal/platform/config"

	redis_pkg "github.com/redis/go-redis/v9"
)

func RegisterConnectRedis(ctx context.Context, cfg config.ConfigRedisEnv) (*redis_pkg.Client, error) {
	client := redis_pkg.NewClient(&redis_pkg.Options{
		Addr:     cfg.Host,
		Password: cfg.Password,
		DB:       cfg.Db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Ping connect redis failed: %w", err)
	}

	return client, nil
}
