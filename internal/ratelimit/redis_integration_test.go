//go:build integration

package ratelimit

import (
	"context"
	"github.com/redis/go-redis/v9"
	"os"
	"testing"
	"time"
)

func TestRedisLimiterScopesCountersByTenant(t *testing.T) {
	c := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})
	if os.Getenv("REDIS_ADDR") == "" {
		t.Fatal("REDIS_ADDR required")
	}
	defer c.Close()
	if err := c.FlushDB(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	l := Redis{Client: c, Limit: 1, Window: time.Minute}
	ok, err := l.Allow(context.Background(), "a")
	if err != nil || !ok {
		t.Fatalf("first %v %v", ok, err)
	}
	ok, err = l.Allow(context.Background(), "a")
	if err != nil || ok {
		t.Fatalf("second %v %v", ok, err)
	}
	ok, err = l.Allow(context.Background(), "b")
	if err != nil || !ok {
		t.Fatalf("other tenant %v %v", ok, err)
	}
}
