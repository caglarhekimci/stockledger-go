package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/caglarhekimci/stockledger-go/internal/app"
	"github.com/caglarhekimci/stockledger-go/internal/domain"
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
		if err = rows.Scan(&i.ID, &i.TenantID, &i.SKU, &i.Available); err != nil {
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
	res, err := domain.NewReservation(in.TenantID, in.ItemID, in.Quantity)
	if err != nil {
		return res, false, err
	}
	// This claim is a serialization point. A racing request waits on the unique
	// index and then reads the committed winner rather than leaking a SQL conflict.
	var claimed string
	err = tx.QueryRow(ctx, `INSERT INTO idempotency_keys(tenant_id,key,payload_hash,reservation_id) VALUES($1,$2,$3,NULL) ON CONFLICT DO NOTHING RETURNING payload_hash`, in.TenantID, in.IdempotencyKey, h).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		var id, existingHash string
		if err = tx.QueryRow(ctx, `SELECT reservation_id,payload_hash FROM idempotency_keys WHERE tenant_id=$1 AND key=$2 FOR UPDATE`, in.TenantID, in.IdempotencyKey).Scan(&id, &existingHash); err != nil {
			return res, false, err
		}
		if existingHash != h {
			return res, false, ErrIdempotencyConflict
		}
		replay, err := reservationByID(ctx, tx, in.TenantID, id, false)
		return replay, true, err
	}
	if err != nil {
		return res, false, err
	}
	tag, err := tx.Exec(ctx, `UPDATE stock_items SET available=available-$1,updated_at=now() WHERE id=$2 AND tenant_id=$3 AND available >= $1`, in.Quantity, in.ItemID, in.TenantID)
	if err != nil {
		return res, false, err
	}
	if tag.RowsAffected() != 1 {
		return res, false, ErrInsufficientStock
	}
	if _, err = tx.Exec(ctx, `INSERT INTO reservations(id,tenant_id,item_id,quantity,status) VALUES($1,$2,$3,$4,$5)`, res.ID, res.TenantID, res.ItemID, res.Quantity, res.Status); err != nil {
		return res, false, err
	}
	if _, err = tx.Exec(ctx, `UPDATE idempotency_keys SET reservation_id=$3 WHERE tenant_id=$1 AND key=$2`, in.TenantID, in.IdempotencyKey, res.ID); err != nil {
		return res, false, err
	}
	eventID := uuid.NewString()
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,type,payload) VALUES($1,$2,'reservation.created',jsonb_build_object('reservation_id',$3::text,'item_id',$4::text,'quantity',$5::integer))`, eventID, in.TenantID, res.ID, in.ItemID, in.Quantity); err != nil {
		return res, false, err
	}
	return res, false, tx.Commit(ctx)
}
func reservationByID(ctx context.Context, q pgx.Tx, t, id string, lock bool) (domain.Reservation, error) {
	var r domain.Reservation
	sql := `SELECT id,tenant_id,item_id,quantity,status,created_at FROM reservations WHERE tenant_id=$1 AND id=$2`
	if lock {
		sql += ` FOR UPDATE`
	}
	err := q.QueryRow(ctx, sql, t, id).Scan(&r.ID, &r.TenantID, &r.ItemID, &r.Quantity, &r.Status, &r.CreatedAt)
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
	res, err := reservationByID(ctx, tx, t, id, true)
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
	if _, err = tx.Exec(ctx, `UPDATE reservations SET status=$1,updated_at=now() WHERE id=$2 AND tenant_id=$3`, res.Status, id, t); err != nil {
		return res, err
	}
	if to == domain.ReservationReleased {
		if _, err = tx.Exec(ctx, `UPDATE stock_items SET available=available+$1,updated_at=now() WHERE id=$2 AND tenant_id=$3`, res.Quantity, res.ItemID, t); err != nil {
			return res, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(id,tenant_id,type,payload) VALUES($1,$2,$3,jsonb_build_object('reservation_id',$4::text))`, uuid.NewString(), t, "reservation."+string(to), id); err != nil {
		return res, err
	}
	return res, tx.Commit(ctx)
}

// ClaimOutbox leases a bounded batch in one statement. A crashed publisher can
// be retried after the lease expires; late workers cannot acknowledge it.
func (r Repository) ClaimOutbox(ctx context.Context, limit int, lease time.Duration) ([]app.OutboxEvent, error) {
	token := uuid.NewString()
	rows, err := r.Pool.Query(ctx, `WITH candidates AS (SELECT id FROM outbox_events WHERE published_at IS NULL AND next_attempt_at<=now() AND (lease_until IS NULL OR lease_until<now()) ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1), claimed AS (UPDATE outbox_events e SET lease_token=$2,lease_until=now()+make_interval(secs=>$3),attempts=attempts+1 FROM candidates c WHERE e.id=c.id RETURNING e.id,e.tenant_id,e.type,e.payload::text,e.lease_token) SELECT id,tenant_id,type,payload,lease_token FROM claimed`, limit, token, lease.Seconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []app.OutboxEvent
	for rows.Next() {
		var e app.OutboxEvent
		if err = rows.Scan(&e.ID, &e.TenantID, &e.Type, &e.Payload, &e.LeaseToken); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r Repository) MarkPublished(ctx context.Context, id, token string) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE outbox_events SET published_at=now(),lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_token=$2 AND published_at IS NULL`, id, token)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errors.New("outbox lease lost")
	}
	return nil
}
func (r Repository) RecordFailure(ctx context.Context, id, token string, delay time.Duration) error {
	_, err := r.Pool.Exec(ctx, `UPDATE outbox_events SET lease_token=NULL,lease_until=NULL,next_attempt_at=now()+make_interval(secs=>$3) WHERE id=$1 AND lease_token=$2 AND published_at IS NULL`, id, token, delay.Seconds())
	return err
}
