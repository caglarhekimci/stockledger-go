package httpapi

import (
	"context"
	"github.com/cags/stockledger/internal/app"
	"github.com/cags/stockledger/internal/domain"
	"net/http"
	"net/http/httptest"
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
