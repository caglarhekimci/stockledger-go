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

var allowScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[2]) end
if n <= tonumber(ARGV[1]) then return 1 end
return 0`)

func (r Redis) Allow(ctx context.Context, tenant string) (bool, error) {
	key := "stockledger:rate:" + tenant
	result, err := allowScript.Run(ctx, r.Client, []string{key}, r.Limit, r.Window.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
func (r Redis) String() string { return fmt.Sprintf("redis limiter (%d/%s)", r.Limit, r.Window) }
