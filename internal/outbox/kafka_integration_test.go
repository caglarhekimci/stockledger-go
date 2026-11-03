//go:build integration

package outbox

import (
	"context"
	"github.com/IBM/sarama"
	"github.com/cags/stockledger/internal/app"
	"os"
	"testing"
	"time"
)

func TestKafkaPublisherPublishesStableEventID(t *testing.T) {
	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		t.Fatal("KAFKA_BROKERS required")
	}
	topic := "stockledger-integration"
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_9_0_0
	cfg.Producer.Return.Successes = true
	cfg.Consumer.Return.Errors = true
	p, err := sarama.NewSyncProducer([]string{broker}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := (Kafka{Producer: p, Topic: topic}).Publish(context.Background(), app.OutboxEvent{ID: "event-1", TenantID: "tenant-a", Type: "reservation.created", Payload: `{"id":"event-1"}`}); err != nil {
		t.Fatal(err)
	}
	consumer, err := sarama.NewConsumer([]string{broker}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer consumer.Close()
	part, err := consumer.ConsumePartition(topic, 0, sarama.OffsetOldest)
	if err != nil {
		t.Fatal(err)
	}
	defer part.Close()
	select {
	case m := <-part.Messages():
		if string(m.Key) != "event-1" {
			t.Fatalf("key %s", m.Key)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("no Kafka message")
	}
}
