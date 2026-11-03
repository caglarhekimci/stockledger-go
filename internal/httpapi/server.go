package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cags/stockledger/internal/app"
	"github.com/cags/stockledger/internal/domain"
	"github.com/cags/stockledger/internal/metrics"
	"github.com/cags/stockledger/internal/postgres"
	"github.com/cags/stockledger/internal/ratelimit"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	Service app.Service
	Keys    map[string]string
	Limiter ratelimit.Limiter
	Ready   func(context.Context) error
}

func (s Server) Router() http.Handler {
	r := chi.NewRouter()
	// chi middleware is global to the route tree, so install it before the
	// unauthenticated probes and isolate authentication to API routes.
	r.Use(metrics.Middleware)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.Ready != nil && s.Ready(r.Context()) != nil {
			writeErr(w, http.StatusServiceUnavailable, "dependencies unavailable")
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})
	r.Handle("/metrics", metrics.Handler())
	r.Group(func(api chi.Router) {
		api.Use(s.auth)
		api.Get("/v1/items", s.list)
		api.Post("/v1/items", s.create)
		api.Post("/v1/reservations", s.reserve)
		api.Post("/v1/reservations/{id}/confirm", s.confirm)
		api.Post("/v1/reservations/{id}/release", s.release)
	})
	return r
}

type ctxKey struct{}

func (s Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		tenant, ok := s.Keys[key]
		if !ok || key == "" {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		ok, err := s.Limiter.Allow(r.Context(), tenant)
		if err != nil {
			writeErr(w, 503, "rate limiter unavailable")
			return
		}
		if !ok {
			writeErr(w, 429, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, tenant)))
	})
}
func tenant(r *http.Request) string { return r.Context().Value(ctxKey{}).(string) }
func (s Server) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SKU       string `json:"sku"`
		Available int    `json:"available"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.SKU) == "" {
		writeErr(w, 400, "sku required")
		return
	}
	if err := s.Service.CreateItem(r.Context(), app.Item{TenantID: tenant(r), SKU: in.SKU, Available: in.Available}); err != nil {
		writeErr(w, 400, err.Error())
		return
	}
	w.WriteHeader(201)
}
func (s Server) list(w http.ResponseWriter, r *http.Request) {
	x, err := s.Service.ListItems(r.Context(), tenant(r))
	if err != nil {
		writeErr(w, 500, "internal error")
		return
	}
	json.NewEncoder(w).Encode(x)
}
func (s Server) reserve(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
	}
	if !decode(w, r, &in) {
		return
	}
	res, replay, err := s.Service.Reserve(r.Context(), app.ReserveInput{TenantID: tenant(r), ItemID: in.ItemID, Quantity: in.Quantity, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		status := 400
		if errors.Is(err, postgres.ErrInsufficientStock) {
			status = 409
		}
		if errors.Is(err, postgres.ErrIdempotencyConflict) {
			status = 409
		}
		writeErr(w, status, err.Error())
		return
	}
	w.Header().Set("Idempotent-Replay", map[bool]string{true: "true", false: "false"}[replay])
	w.WriteHeader(201)
	json.NewEncoder(w).Encode(res)
}
func (s Server) transition(to domain.ReservationStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var res domain.Reservation
		var err error
		if to == domain.ReservationConfirmed {
			res, err = s.Service.Confirm(r.Context(), tenant(r), chi.URLParam(r, "id"))
		} else {
			res, err = s.Service.Release(r.Context(), tenant(r), chi.URLParam(r, "id"))
		}
		if err != nil {
			writeErr(w, 409, err.Error())
			return
		}
		json.NewEncoder(w).Encode(res)
	}
}
func (s Server) confirm(w http.ResponseWriter, r *http.Request) {
	s.transition(domain.ReservationConfirmed)(w, r)
}
func (s Server) release(w http.ResponseWriter, r *http.Request) {
	s.transition(domain.ReservationReleased)(w, r)
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, 400, "invalid json")
		return false
	}
	return true
}
func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

var _ = time.Second
