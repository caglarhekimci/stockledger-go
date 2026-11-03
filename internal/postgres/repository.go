package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cags/stockledger/internal/app"
	"github.com/cags/stockledger/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

var (
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrIdempotencyConflict = errors.New("idempotency key payload conflict")
	ErrNotFound            = errors.New("not found")
)

type Repository struct{ Pool *pgxpool.Pool }

func (r Repository) CreateItem(ctx context.Context, i app.Item) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	_, err := r.Pool.Exec(ctx, `INSERT INTO stock_items(id,tenant_id,sku,available) VALUES($1,$2,$3,$4)`, i.ID, i.TenantID, i.SKU, i.Available)
	return err
}
func (r Repository) ListItems(ctx context.Context, t string) ([]app.Item, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,tenant_id,sku,available FROM stock_items WHERE tenant_id=$1 ORDER BY sku`, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []app.Item{}
	for rows.Next() {
		var i app.Item
		if err := rows.Scan(&i.ID, &i.TenantID, &i.SKU, &i.Available); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}
func hash(in app.ReserveInput) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", in.ItemID, in.Quantity)))
	return hex.EncodeToString(h[:])
}
func (r Repository) Reserve(ctx context.Context, in app.ReserveInput) (domain.Reservation, bool, error) {
	tx, err := r.Pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.Reservation{}, false, err
	}
	defer tx.Rollback(ctx)
	h := hash(in)
	var existingID, existingHash string
	err = tx.QueryRow(ctx, `SELECT reservation_id,payload_hash FROM idempotency_keys WHERE tenant_id=$1 AND key=$2 FOR UPDATE`, in.TenantID, in.IdempotencyKey).Scan(&existingID, &existingHash)
	if err == nil {
		if existingHash != h {
			return domain.Reservation{}, false, ErrIdempotencyConflict
		}
		res, err := reservationByID(ctx, tx, in.TenantID, existingID)
		return res, true, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Reservation{}, false, err
	}
	res, err := domain.NewReservation(in.TenantID, in.ItemID, in.Quantity)
	if err != nil {
		return res, false, err
	}
	// Claim the key before changing stock. A concurrent identical request blocks on
	// the primary key, then observes this committed reservation as a replay.
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(tenant_id,key,payload_hash,reservation_id) VALUES($1,$2,$3,NULL)`, in.TenantID, in.IdempotencyKey, h)
	if err != nil {
		return res, false, err
	}
	tag, err := tx.Exec(ctx, `UPDATE stock_items SET available=available-$1, updated_at=now() WHERE id=$2 AND tenant_id=$3 AND available >= $1`, in.Quantity, in.ItemID, in.TenantID)
	if err != nil {
		return res, false, err
	}
	if tag.RowsAffected() != 1 {
		return res, false, ErrInsufficientStock
	}
	_, err = tx.Exec(ctx, `INSERT INTO reservations(id,tenant_id,item_id,quantity,status) VALUES($1,$2,$3,$4,$5)`, res.ID, res.TenantID, res.ItemID, res.Quantity, res.Status)
	if err != nil {
		return res, false, err
	}
	_, err = tx.Exec(ctx, `UPDATE idempotency_keys SET reservation_id=$3 WHERE tenant_id=$1 AND key=$2`, in.TenantID, in.IdempotencyKey, res.ID)
	if err != nil {
		return res, false, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,type,payload) VALUES($1,$2,'reservation.created',jsonb_build_object('reservation_id',$1,'item_id',$3,'quantity',$4))`, uuid.NewString(), in.TenantID, in.ItemID, in.Quantity)
	if err != nil {
		return res, false, err
	}
	return res, false, tx.Commit(ctx)
}
func reservationByID(ctx context.Context, q pgx.Tx, t, id string) (domain.Reservation, error) {
	var r domain.Reservation
	err := q.QueryRow(ctx, `SELECT id,tenant_id,item_id,quantity,status,created_at FROM reservations WHERE tenant_id=$1 AND id=$2`, t, id).Scan(&r.ID, &r.TenantID, &r.ItemID, &r.Quantity, &r.Status, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}
func (r Repository) Transition(ctx context.Context, t, id string, to domain.ReservationStatus) (domain.Reservation, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return domain.Reservation{}, err
	}
	defer tx.Rollback(ctx)
	res, err := reservationByID(ctx, tx, t, id)
	if err != nil {
		return res, err
	}
	if to == domain.ReservationConfirmed {
		err = res.Confirm()
	} else {
		err = res.Release()
	}
	if err != nil {
		return res, err
	}
	_, err = tx.Exec(ctx, `UPDATE reservations SET status=$1,updated_at=now() WHERE id=$2 AND tenant_id=$3`, res.Status, id, t)
	if err != nil {
		return res, err
	}
	if to == domain.ReservationReleased {
		_, err = tx.Exec(ctx, `UPDATE stock_items SET available=available+$1,updated_at=now() WHERE id=$2 AND tenant_id=$3`, res.Quantity, res.ItemID, t)
		if err != nil {
			return res, err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,type,payload) VALUES($1,$2,$3,jsonb_build_object('reservation_id',$4))`, uuid.NewString(), t, "reservation."+string(to), id)
	if err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}

type OutboxEvent struct {
	ID, TenantID, Type, Payload string
	CreatedAt                   time.Time
}

func (r Repository) NextOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id,tenant_id,type,payload::text,created_at FROM outbox_events WHERE published_at IS NULL ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err = rows.Scan(&e.ID, &e.TenantID, &e.Type, &e.Payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r Repository) MarkPublished(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, `UPDATE outbox_events SET published_at=now(),attempts=attempts+1 WHERE id=$1 AND published_at IS NULL`, id)
	return err
}
