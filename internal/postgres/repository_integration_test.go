//go:build integration

package postgres

import (
	"context"
	"github.com/cags/stockledger/internal/app"
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
