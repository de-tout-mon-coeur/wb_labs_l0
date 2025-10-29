CREATE TABLE IF NOT EXISTS orders (
  order_uid TEXT PRIMARY KEY,
  track_number TEXT,
  data JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_track ON orders(track_number);
