// Package app coordinates use cases without depending on transport or infrastructure.
package app

import (
	"context"
	"errors"
	"github.com/cags/stockledger/internal/domain"
)

var ErrIdempotencyKeyRequired = errors.New("idempotency key is required")

type Item struct {
	ID, TenantID, SKU string
	Available         int
}
type ReserveInput struct {
	TenantID, ItemID, IdempotencyKey string
	Quantity                         int
}
type Repository interface {
	CreateItem(context.Context, Item) error
	ListItems(context.Context, string) ([]Item, error)
	Reserve(context.Context, ReserveInput) (domain.Reservation, bool, error)
	Transition(context.Context, string, string, domain.ReservationStatus) (domain.Reservation, error)
}
type Service struct{ Repo Repository }

func (s Service) CreateItem(ctx context.Context, item Item) error {
	if item.Available < 0 {
		return errors.New("available stock cannot be negative")
	}
	return s.Repo.CreateItem(ctx, item)
}
func (s Service) ListItems(ctx context.Context, tenant string) ([]Item, error) {
	return s.Repo.ListItems(ctx, tenant)
}
func (s Service) Reserve(ctx context.Context, in ReserveInput) (domain.Reservation, bool, error) {
	if in.IdempotencyKey == "" {
		return domain.Reservation{}, false, ErrIdempotencyKeyRequired
	}
	return s.Repo.Reserve(ctx, in)
}
func (s Service) Confirm(ctx context.Context, tenant, id string) (domain.Reservation, error) {
	return s.Repo.Transition(ctx, tenant, id, domain.ReservationConfirmed)
}
func (s Service) Release(ctx context.Context, tenant, id string) (domain.Reservation, error) {
	return s.Repo.Transition(ctx, tenant, id, domain.ReservationReleased)
}
