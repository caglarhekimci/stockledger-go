package outbox

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/cags/stockledger/internal/postgres"
	"time"
)

type Publisher interface {
	Publish(context.Context, postgres.OutboxEvent) error
	Close() error
}
type Kafka struct {
	Producer sarama.SyncProducer
	Topic    string
}

func (k Kafka) Publish(ctx context.Context, e postgres.OutboxEvent) error {
	_, _, err := k.Producer.SendMessage(&sarama.ProducerMessage{Topic: k.Topic, Key: sarama.StringEncoder(e.ID), Value: sarama.StringEncoder(e.Payload), Headers: []sarama.RecordHeader{{Key: []byte("event_type"), Value: []byte(e.Type)}, {Key: []byte("tenant_id"), Value: []byte(e.TenantID)}}})
	return err
}
func (k Kafka) Close() error { return k.Producer.Close() }

type Store interface {
	NextOutbox(context.Context, int) ([]postgres.OutboxEvent, error)
	MarkPublished(context.Context, string) error
}
type Worker struct {
	Store     Store
	Publisher Publisher
	Interval  time.Duration
}

func (w Worker) Run(ctx context.Context) error {
	tick := time.NewTicker(w.Interval)
	defer tick.Stop()
	for {
		if err := w.Once(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
func (w Worker) Once(ctx context.Context) error {
	events, err := w.Store.NextOutbox(ctx, 100)
	if err != nil {
		return err
	}
	for _, e := range events {
		if err = w.Publisher.Publish(ctx, e); err != nil {
			return err
		}
		if err = w.Store.MarkPublished(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}
