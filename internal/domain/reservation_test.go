package domain

import "testing"

func TestNewReservationRejectsNonPositiveQuantity(t *testing.T) {
	_, err := NewReservation("tenant-a", "item-a", 0)
	if err != ErrInvalidQuantity {
		t.Fatalf("got %v, want %v", err, ErrInvalidQuantity)
	}
}

func TestReservationLegalTransitions(t *testing.T) {
	r, err := NewReservation("tenant-a", "item-a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Confirm(); err != nil {
		t.Fatal(err)
	}
	if err := r.Release(); err != ErrInvalidTransition {
		t.Fatalf("got %v", err)
	}
}
