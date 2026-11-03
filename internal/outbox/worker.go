package outbox

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/caglarhekimci/stockledger-go/internal/app"
	"time"
)

type Publisher interface {
	Publish(context.Context, app.OutboxEvent) error
	Close() error
}
type Kafka struct {
	Producer sarama.SyncProducer
	Topic    string
}

func (k Kafka) Publish(ctx context.Context, e app.OutboxEvent) error {
	type result struct{ err error }
	done := make(chan result, 1)
	go func() {
		_, _, err := k.Producer.SendMessage(&sarama.ProducerMessage{Topic: k.Topic, Key: sarama.StringEncoder(e.ID), Value: sarama.StringEncoder(e.Payload), Headers: []sarama.RecordHeader{{Key: []byte("event_type"), Value: []byte(e.Type)}, {Key: []byte("tenant_id"), Value: []byte(e.TenantID)}}})
		done <- result{err}
	}()
	select {
	case x := <-done:
		return x.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (k Kafka) Close() error { return k.Producer.Close() }

type Store interface {
	ClaimOutbox(context.Context, int, time.Duration) ([]app.OutboxEvent, error)
	MarkPublished(context.Context, string, string) error
	RecordFailure(context.Context, string, string, time.Duration) error
}
type Worker struct {
	Store                                       Store
	Publisher                                   Publisher
	Interval, Lease, RetryDelay, PublishTimeout time.Duration
}

func (w Worker) defaults() Worker {
	if w.Interval <= 0 {
		w.Interval = time.Second
	}
	if w.Lease <= 0 {
		w.Lease = 30 * time.Second
	}
	if w.RetryDelay <= 0 {
		w.RetryDelay = time.Second
	}
	if w.PublishTimeout <= 0 {
		w.PublishTimeout = 5 * time.Second
	}
	return w
}
func (w Worker) Run(ctx context.Context) error {
	w = w.defaults()
	tick := time.NewTicker(w.Interval)
	defer tick.Stop()
	for {
		if err := w.Once(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.RetryDelay):
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
func (w Worker) Once(ctx context.Context) error {
	w = w.defaults()
	// Claim one event at a time: sequential publication can never consume a
	// later event's lease. Throughput can be increased safely with more workers.
	events, err := w.Store.ClaimOutbox(ctx, 1, w.Lease)
	if err != nil {
		return err
	}
	for _, e := range events {
		publishCtx, cancel := context.WithTimeout(ctx, w.PublishTimeout)
		err = w.Publisher.Publish(publishCtx, e)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if recordErr := w.Store.RecordFailure(ctx, e.ID, e.LeaseToken, w.RetryDelay); recordErr != nil {
				return recordErr
			}
			continue
		}
		if err = w.Store.MarkPublished(ctx, e.ID, e.LeaseToken); err != nil {
			return err
		}
	}
	return nil
}
