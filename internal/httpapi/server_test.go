package httpapi

import (
	"context"
	"errors"
	"github.com/caglarhekimci/stockledger-go/internal/app"
	"github.com/caglarhekimci/stockledger-go/internal/domain"
	"github.com/caglarhekimci/stockledger-go/internal/postgres"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memoryRepo struct{}

func (memoryRepo) CreateItem(context.Context, app.Item) error            { return nil }
func (memoryRepo) ListItems(context.Context, string) ([]app.Item, error) { return nil, nil }
func (memoryRepo) Reserve(context.Context, app.ReserveInput) (domain.Reservation, bool, error) {
	return domain.Reservation{}, false, nil
}
func (memoryRepo) Transition(context.Context, string, string, domain.ReservationStatus) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

type permit struct{}

func (permit) Allow(context.Context, string) (bool, error) { return true, nil }
func TestRejectsRequestsWithoutAPIKey(t *testing.T) {
	s := Server{Service: app.Service{Repo: memoryRepo{}}, Keys: map[string]string{"good": "tenant-a"}, Limiter: permit{}}
	r := httptest.NewRequest(http.MethodGet, "/v1/items", nil)
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", w.Code)
	}
}

type errorRepo struct {
	memoryRepo
	reserveErr, transitionErr error
}

func (e errorRepo) Reserve(context.Context, app.ReserveInput) (domain.Reservation, bool, error) {
	return domain.Reservation{}, false, e.reserveErr
}
func (e errorRepo) Transition(context.Context, string, string, domain.ReservationStatus) (domain.Reservation, error) {
	return domain.Reservation{}, e.transitionErr
}
func TestClassifiesNotFoundAndDoesNotLeakInfrastructureErrors(t *testing.T) {
	base := map[string]string{"good": "tenant-a"}
	s := Server{Service: app.Service{Repo: errorRepo{transitionErr: postgres.ErrNotFound}}, Keys: base, Limiter: permit{}}
	r := httptest.NewRequest(http.MethodPost, "/v1/reservations/missing/confirm", nil)
	r.Header.Set("X-API-Key", "good")
	w := httptest.NewRecorder()
	s.Router().ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("notfound=%d", w.Code)
	}
	s = Server{Service: app.Service{Repo: errorRepo{reserveErr: errors.New("pq: private server detail")}}, Keys: base, Limiter: permit{}}
	r = httptest.NewRequest(http.MethodPost, "/v1/reservations", strings.NewReader(`{"item_id":"x","quantity":1}`))
	r.Header.Set("X-API-Key", "good")
	r.Header.Set("Idempotency-Key", "k")
	w = httptest.NewRecorder()
	s.Router().ServeHTTP(w, r)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "private") {
		t.Fatalf("status/body %d %s", w.Code, w.Body.String())
	}
}
