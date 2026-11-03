//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cags/stockledger/internal/app"
	"github.com/cags/stockledger/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func integrationRepo(t *testing.T) Repository {
	t.Helper()
	u := os.Getenv("DATABASE_URL")
	if u == "" {
		t.Fatal("DATABASE_URL required for integration tests")
	}
	p, err := pgxpool.New(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	sql, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Exec(context.Background(), "DROP TABLE IF EXISTS outbox_events,idempotency_keys,reservations,stock_items CASCADE;"+string(sql))
	if err != nil {
		t.Fatal(err)
	}
	return Repository{Pool: p}
}
func TestReserveIsAtomicIdempotentAndTenantScoped(t *testing.T) {
	r := integrationRepo(t)
	ctx := context.Background()
	if err := r.CreateItem(ctx, app.Item{ID: "11111111-1111-1111-1111-111111111111", TenantID: "a", SKU: "widget", Available: 5}); err != nil {
		t.Fatal(err)
	}
	in := app.ReserveInput{TenantID: "a", ItemID: "11111111-1111-1111-1111-111111111111", Quantity: 2, IdempotencyKey: "same"}
	first, replay, err := r.Reserve(ctx, in)
	if err != nil || replay {
		t.Fatalf("first: %v %v", replay, err)
	}
	again, replay, err := r.Reserve(ctx, in)
	if err != nil || !replay || again.ID != first.ID {
		t.Fatalf("replay %v %v %#v", replay, err, again)
	}
	if _, _, err = r.Reserve(ctx, app.ReserveInput{TenantID: "b", ItemID: in.ItemID, Quantity: 1, IdempotencyKey: "x"}); err == nil {
		t.Fatal("cross tenant reservation succeeded")
	}
}
func TestConcurrentReservationsNeverOversell(t *testing.T) {
	r := integrationRepo(t)
	ctx := context.Background()
	id := "22222222-2222-2222-2222-222222222222"
	if err := r.CreateItem(ctx, app.Item{ID: id, TenantID: "a", SKU: "limited", Available: 5}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _, err := r.Reserve(ctx, app.ReserveInput{TenantID: "a", ItemID: id, Quantity: 1, IdempotencyKey: string(rune(n + 65))})
			if err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(n)
	}
	wg.Wait()
	if ok != 5 {
		t.Fatalf("successful reservations %d want 5", ok)
	}
}

func TestSameKeyConcurrentReserveReplaysOnceAndWritesOneOutboxEvent(t *testing.T) {
	r := integrationRepo(t)
	ctx := context.Background()
	id := "33333333-3333-3333-3333-333333333333"
	if err := r.CreateItem(ctx, app.Item{ID: id, TenantID: "a", SKU: "same-key", Available: 5}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	type result struct {
		replay bool
		id     string
		err    error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			<-start
			res, replay, err := r.Reserve(ctx, app.ReserveInput{TenantID: "a", ItemID: id, Quantity: 1, IdempotencyKey: "one"})
			results <- result{replay, res.ID, err}
		}()
	}
	close(start)
	a, b := <-results, <-results
	if a.err != nil || b.err != nil || a.id != b.id || a.replay == b.replay {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
	var stock, count int
	if err := r.Pool.QueryRow(ctx, `SELECT available FROM stock_items WHERE id=$1`, id).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if err := r.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox_events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if stock != 4 || count != 1 {
		t.Fatalf("stock=%d events=%d", stock, count)
	}
	var payload []byte
	if err := r.Pool.QueryRow(ctx, `SELECT payload FROM outbox_events`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatal(err)
	}
	if body["reservation_id"] != a.id {
		t.Fatalf("payload reservation_id=%v want %s", body["reservation_id"], a.id)
	}
}

func TestConcurrentTransitionsLockReservationAndRestoreOnlyOnce(t *testing.T) {
	r := integrationRepo(t)
	ctx := context.Background()
	id := "44444444-4444-4444-4444-444444444444"
	if err := r.CreateItem(ctx, app.Item{ID: id, TenantID: "a", SKU: "transition", Available: 2}); err != nil {
		t.Fatal(err)
	}
	res, _, err := r.Reserve(ctx, app.ReserveInput{TenantID: "a", ItemID: id, Quantity: 1, IdempotencyKey: "reserve"})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() { <-start; _, e := r.Transition(ctx, "a", res.ID, domain.ReservationReleased); errs <- e }()
	}
	close(start)
	e1, e2 := <-errs, <-errs
	if (e1 == nil) == (e2 == nil) || (!errors.Is(e1, domain.ErrInvalidTransition) && !errors.Is(e2, domain.ErrInvalidTransition)) {
		t.Fatalf("transition errors %v %v", e1, e2)
	}
	var stock int
	if err := r.Pool.QueryRow(ctx, `SELECT available FROM stock_items WHERE id=$1`, id).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 2 {
		t.Fatalf("stock=%d", stock)
	}
	res, _, err = r.Reserve(ctx, app.ReserveInput{TenantID: "a", ItemID: id, Quantity: 1, IdempotencyKey: "reserve-2"})
	if err != nil {
		t.Fatal(err)
	}
	start = make(chan struct{})
	errs = make(chan error, 2)
	go func() { <-start; _, e := r.Transition(ctx, "a", res.ID, domain.ReservationConfirmed); errs <- e }()
	go func() { <-start; _, e := r.Transition(ctx, "a", res.ID, domain.ReservationReleased); errs <- e }()
	close(start)
	e1, e2 = <-errs, <-errs
	if (e1 == nil) == (e2 == nil) || (!errors.Is(e1, domain.ErrInvalidTransition) && !errors.Is(e2, domain.ErrInvalidTransition)) {
		t.Fatalf("confirm/release errors %v %v", e1, e2)
	}
}
