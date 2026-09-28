// Package cache wraps the shared Redis instance used as a best-effort cache.
// Callers treat every error as a cache miss and fall back to the source.
package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrMiss is returned by Get when the key does not exist.
var ErrMiss = errors.New("cache miss")

// opTimeout bounds each Redis call so a slow or down Redis never stalls a
// request for longer than a fallback would take.
const opTimeout = 200 * time.Millisecond

type Redis struct {
	rdb *redis.Client
}

func NewRedis(addr string, db int) *Redis {
	return &Redis{rdb: redis.NewClient(&redis.Options{Addr: addr, DB: db})}
}

func (r *Redis) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	return r.rdb.Ping(ctx).Err()
}

func (r *Redis) Get(ctx context.Context, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	b, err := r.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrMiss
	}
	return b, err
}

func (r *Redis) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	return r.rdb.Set(ctx, key, val, ttl).Err()
}

func (r *Redis) Close() error {
	return r.rdb.Close()
}
