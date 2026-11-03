package ratelimit

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"time"
)

type Limiter interface {
	Allow(context.Context, string) (bool, error)
}
type Redis struct {
	Client *redis.Client
	Limit  int64
	Window time.Duration
}

func (r Redis) Allow(ctx context.Context, tenant string) (bool, error) {
	key := "stockledger:rate:" + tenant
	n, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err = r.Client.Expire(ctx, key, r.Window).Err(); err != nil {
			return false, err
		}
	}
	return n <= r.Limit, nil
}
func (r Redis) String() string { return fmt.Sprintf("redis limiter (%d/%s)", r.Limit, r.Window) }
