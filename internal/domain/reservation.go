package domain

import (
	"errors"
	"github.com/google/uuid"
	"time"
)

var (
	ErrInvalidQuantity   = errors.New("quantity must be positive")
	ErrInvalidTransition = errors.New("invalid reservation state transition")
)

type ReservationStatus string

const (
	ReservationPending   ReservationStatus = "pending"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationReleased  ReservationStatus = "released"
)

type Reservation struct {
	ID        string            `json:"id"`
	TenantID  string            `json:"tenant_id"`
	ItemID    string            `json:"item_id"`
	Quantity  int               `json:"quantity"`
	Status    ReservationStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}

func NewReservation(tenantID, itemID string, quantity int) (Reservation, error) {
	if quantity <= 0 {
		return Reservation{}, ErrInvalidQuantity
	}
	return Reservation{ID: uuid.NewString(), TenantID: tenantID, ItemID: itemID, Quantity: quantity, Status: ReservationPending, CreatedAt: time.Now().UTC()}, nil
}
func (r *Reservation) Confirm() error {
	if r.Status != ReservationPending {
		return ErrInvalidTransition
	}
	r.Status = ReservationConfirmed
	return nil
}
func (r *Reservation) Release() error {
	if r.Status != ReservationPending {
		return ErrInvalidTransition
	}
	r.Status = ReservationReleased
	return nil
}
