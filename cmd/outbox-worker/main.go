package main

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/caglarhekimci/stockledger-go/internal/outbox"
	"github.com/caglarhekimci/stockledger-go/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db := mustEnv("DATABASE_URL")
	pool, err := pgxpool.New(ctx, db)
	if err != nil {
		panic(err)
	}
	defer pool.Close()
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_9_0_0
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Retry.Max = 3
	cfg.Producer.Retry.Backoff = 250 * time.Millisecond
	cfg.Producer.Return.Successes = true
	cfg.Net.DialTimeout = 5 * time.Second
	cfg.Net.ReadTimeout = 5 * time.Second
	cfg.Net.WriteTimeout = 5 * time.Second
	producer, err := sarama.NewSyncProducer([]string{env("KAFKA_BROKERS", "localhost:9092")}, cfg)
	if err != nil {
		panic(err)
	}
	worker := outbox.Worker{Store: postgres.Repository{Pool: pool}, Publisher: outbox.Kafka{Producer: producer, Topic: env("KAFKA_TOPIC", "stockledger.events")}, Interval: time.Second}
	slog.Info("outbox worker started")
	if err := worker.Run(ctx); err != nil && err != context.Canceled {
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
