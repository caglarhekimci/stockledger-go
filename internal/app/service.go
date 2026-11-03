// Package app coordinates use cases without depending on transport or infrastructure.
package app

import (
	"context"
	"errors"
	"github.com/caglarhekimci/stockledger-go/internal/domain"
)

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrInvalidStock           = errors.New("available stock cannot be negative")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrIdempotencyConflict    = errors.New("idempotency key payload conflict")
	ErrNotFound               = errors.New("not found")
)

type Item struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	SKU       string `json:"sku"`
	Available int    `json:"available"`
}
type ReserveInput struct {
	TenantID, ItemID, IdempotencyKey string
	Quantity                         int
}

// OutboxEvent is a transport-neutral, leased message. Its ID is the stable
// consumer-deduplication key; LeaseToken prevents a stale worker acknowledging it.
type OutboxEvent struct{ ID, TenantID, Type, Payload, LeaseToken string }
type Repository interface {
	CreateItem(context.Context, Item) error
	ListItems(context.Context, string) ([]Item, error)
	Reserve(context.Context, ReserveInput) (domain.Reservation, bool, error)
	Transition(context.Context, string, string, domain.ReservationStatus) (domain.Reservation, error)
}
type Service struct{ Repo Repository }

func (s Service) CreateItem(ctx context.Context, item Item) error {
	if item.Available < 0 {
		return ErrInvalidStock
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
