package main

import (
	"context"
	"github.com/cags/stockledger/internal/app"
	"github.com/cags/stockledger/internal/httpapi"
	"github.com/cags/stockledger/internal/postgres"
	"github.com/cags/stockledger/internal/ratelimit"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dbURL := mustEnv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "localhost:6379")})
	defer redisClient.Close()
	keys := map[string]string{}
	for _, pair := range strings.Split(os.Getenv("API_KEYS"), ",") {
		p := strings.SplitN(pair, ":", 2)
		if len(p) == 2 {
			keys[p[0]] = p[1]
		}
	}
	repo := postgres.Repository{Pool: pool}
	srv := httpapi.Server{Service: app.Service{Repo: repo}, Keys: keys, Limiter: ratelimit.Redis{Client: redisClient, Limit: 100, Window: time.Minute}, Ready: func(c context.Context) error {
		if err := pool.Ping(c); err != nil {
			return err
		}
		return redisClient.Ping(c).Err()
	}}
	server := &http.Server{Addr: env("HTTP_ADDR", ":8080"), Handler: srv.Router(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()
	slog.Info("stockledger listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		panic(k + " is required")
	}
	return v
}
