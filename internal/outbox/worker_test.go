package outbox

import (
	"context"
	"errors"
	"github.com/caglarhekimci/stockledger-go/internal/app"
	"testing"
	"time"
)

type fakeStore struct {
	events         []app.OutboxEvent
	acks, failures int
	maxClaim       int
}

func (s *fakeStore) ClaimOutbox(_ context.Context, limit int, _ time.Duration) ([]app.OutboxEvent, error) {
	s.maxClaim = limit
	return s.events, nil
}
func (s *fakeStore) MarkPublished(context.Context, string, string) error { s.acks++; return nil }
func (s *fakeStore) RecordFailure(context.Context, string, string, time.Duration) error {
	s.failures++
	return nil
}

type flakyPublisher struct{ calls int }

func (p *flakyPublisher) Publish(context.Context, app.OutboxEvent) error {
	p.calls++
	if p.calls == 1 {
		return errors.New("temporary")
	}
	return nil
}
func (*flakyPublisher) Close() error { return nil }
func TestOnceDoesNotAckFailureAndRetriesNextPass(t *testing.T) {
	s := &fakeStore{events: []app.OutboxEvent{{ID: "e", LeaseToken: "l"}}}
	p := &flakyPublisher{}
	w := Worker{Store: s, Publisher: p, RetryDelay: time.Millisecond}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.acks != 0 || s.failures != 1 {
		t.Fatalf("ack=%d failure=%d", s.acks, s.failures)
	}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.acks != 1 || p.calls != 2 {
		t.Fatalf("ack=%d calls=%d", s.acks, p.calls)
	}
}

func TestOnceClaimsOneEventSoLeaseAlwaysCoversPublishBudget(t *testing.T) {
	s := &fakeStore{}
	w := Worker{Store: s, Publisher: &flakyPublisher{}, Lease: 30 * time.Second, PublishTimeout: 5 * time.Second}
	if err := w.Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.maxClaim != 1 {
		t.Fatalf("claimed %d events; sequential publisher must lease one", s.maxClaim)
	}
}
