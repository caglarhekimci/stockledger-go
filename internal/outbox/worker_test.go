package outbox

import (
	"context"
	"errors"
	"github.com/cags/stockledger/internal/app"
	"testing"
	"time"
)

type fakeStore struct {
	events         []app.OutboxEvent
	acks, failures int
}

func (s *fakeStore) ClaimOutbox(context.Context, int, time.Duration) ([]app.OutboxEvent, error) {
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
