package deps

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ConfigDeps struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
}

func NewDeps(pool *pgxpool.Pool, redisClient *redis.Client) *ConfigDeps {
	return &ConfigDeps{
		redisClient: redisClient,
		pool:        pool,
	}
}

func (d *ConfigDeps) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *ConfigDeps) RedisClient() *redis.Client {
	return d.redisClient
}
