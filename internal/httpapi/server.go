package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/caglarhekimci/stockledger-go/internal/app"
	"github.com/caglarhekimci/stockledger-go/internal/domain"
	"github.com/caglarhekimci/stockledger-go/internal/metrics"
	"github.com/caglarhekimci/stockledger-go/internal/ratelimit"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"
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

const maxInt32 = 2147483647

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
	in.SKU = strings.TrimSpace(in.SKU)
	if in.SKU == "" || utf8.RuneCountInString(in.SKU) > 100 || in.Available < 0 || in.Available > maxInt32 {
		writeErr(w, 400, "sku required")
		return
	}
	if err := s.Service.CreateItem(r.Context(), app.Item{TenantID: tenant(r), SKU: in.SKU, Available: in.Available}); err != nil {
		if errors.Is(err, app.ErrInvalidStock) {
			writeErr(w, 400, "invalid stock")
			return
		}
		writeErr(w, 500, "internal error")
		return
	}
	writeJSON(w, 201, map[string]string{"status": "created"})
}
func (s Server) list(w http.ResponseWriter, r *http.Request) {
	x, err := s.Service.ListItems(r.Context(), tenant(r))
	if err != nil {
		writeErr(w, 500, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, x)
}
func (s Server) reserve(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ItemID   string `json:"item_id"`
		Quantity int    `json:"quantity"`
	}
	if !decode(w, r, &in) {
		return
	}
	if _, err := uuid.Parse(in.ItemID); err != nil || in.Quantity < 1 || in.Quantity > maxInt32 {
		writeErr(w, 400, "invalid reservation input")
		return
	}
	if !validIdempotencyKey(r.Header.Get("Idempotency-Key")) {
		writeErr(w, 400, "invalid idempotency key")
		return
	}
	res, replay, err := s.Service.Reserve(r.Context(), app.ReserveInput{TenantID: tenant(r), ItemID: in.ItemID, Quantity: in.Quantity, IdempotencyKey: r.Header.Get("Idempotency-Key")})
	if err != nil {
		if errors.Is(err, app.ErrIdempotencyKeyRequired) {
			writeErr(w, 400, "idempotency key required")
			return
		}
		if errors.Is(err, domain.ErrInvalidQuantity) {
			writeErr(w, 400, "invalid quantity")
			return
		}
		if errors.Is(err, app.ErrInsufficientStock) || errors.Is(err, app.ErrIdempotencyConflict) {
			writeErr(w, 409, "reservation conflict")
			return
		}
		writeErr(w, 500, "internal error")
		return
	}
	w.Header().Set("Idempotent-Replay", map[bool]string{true: "true", false: "false"}[replay])
	writeJSON(w, 201, res)
}
func (s Server) transition(to domain.ReservationStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := uuid.Parse(chi.URLParam(r, "id")); err != nil {
			writeErr(w, 400, "invalid reservation id")
			return
		}
		var res domain.Reservation
		var err error
		if to == domain.ReservationConfirmed {
			res, err = s.Service.Confirm(r.Context(), tenant(r), chi.URLParam(r, "id"))
		} else {
			res, err = s.Service.Release(r.Context(), tenant(r), chi.URLParam(r, "id"))
		}
		if err != nil {
			if errors.Is(err, app.ErrNotFound) {
				writeErr(w, 404, "reservation not found")
				return
			}
			if errors.Is(err, domain.ErrInvalidTransition) {
				writeErr(w, 409, "invalid reservation transition")
				return
			}
			writeErr(w, 500, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, res)
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
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, 400, "invalid json")
		return false
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeErr(w, 400, "invalid json")
		return false
	}
	return true
}
func validIdempotencyKey(key string) bool {
	if len(key) < 1 || len(key) > 128 {
		return false
	}
	for _, r := range key {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == ':') {
			return false
		}
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
