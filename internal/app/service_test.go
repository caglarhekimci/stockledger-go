package app

import (
	"context"
	"github.com/cags/stockledger/internal/domain"
	"testing"
)

type fakeRepo struct{ called bool }

func (f *fakeRepo) CreateItem(context.Context, Item) error            { return nil }
func (f *fakeRepo) ListItems(context.Context, string) ([]Item, error) { return nil, nil }
func (f *fakeRepo) Reserve(_ context.Context, in ReserveInput) (domain.Reservation, bool, error) {
	f.called = true
	return domain.Reservation{ID: "r", TenantID: in.TenantID, ItemID: in.ItemID, Quantity: in.Quantity, Status: domain.ReservationPending}, false, nil
}
func (f *fakeRepo) Transition(context.Context, string, string, domain.ReservationStatus) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}
func TestReserveRejectsBlankIdempotencyKeyBeforeRepository(t *testing.T) {
	f := &fakeRepo{}
	s := Service{Repo: f}
	_, _, err := s.Reserve(context.Background(), ReserveInput{TenantID: "t", ItemID: "i", Quantity: 1})
	if err != ErrIdempotencyKeyRequired {
		t.Fatalf("got %v", err)
	}
	if f.called {
		t.Fatal("repository called")
	}
}
