CREATE TABLE stock_items (id uuid PRIMARY KEY, tenant_id text NOT NULL, sku text NOT NULL, available integer NOT NULL CHECK (available >= 0), updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(tenant_id, sku));
CREATE TABLE reservations (id uuid PRIMARY KEY, tenant_id text NOT NULL, item_id uuid NOT NULL REFERENCES stock_items(id), quantity integer NOT NULL CHECK(quantity > 0), status text NOT NULL CHECK(status IN ('pending','confirmed','released')), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX reservations_tenant_idx ON reservations(tenant_id);
CREATE TABLE idempotency_keys (tenant_id text NOT NULL, key text NOT NULL, payload_hash text NOT NULL, reservation_id uuid REFERENCES reservations(id), created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(tenant_id,key));
CREATE TABLE outbox_events (id uuid PRIMARY KEY, tenant_id text NOT NULL, type text NOT NULL, payload jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), published_at timestamptz, attempts integer NOT NULL DEFAULT 0);
CREATE INDEX outbox_pending_idx ON outbox_events(created_at) WHERE published_at IS NULL;
