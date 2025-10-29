package db

import (
    "database/sql"
    _ "github.com/lib/pq"
    "context"
    "time"
    "encoding/json"
)

type Store struct {
    DB *sql.DB
}

func New(dsn string) (*Store, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil { return nil, err }
    db.SetConnMaxLifetime(time.Minute * 5)
    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    if err := db.Ping(); err != nil { return nil, err }
    s := &Store{DB: db}
    if err := s.ensureSchema(); err != nil { return nil, err }
    return s, nil
}

func (s *Store) ensureSchema() error {
    _, err := s.DB.Exec(`
    CREATE TABLE IF NOT EXISTS orders (
      order_uid TEXT PRIMARY KEY,
      track_number TEXT,
      data JSONB NOT NULL,
      created_at TIMESTAMPTZ DEFAULT now()
    );
    CREATE INDEX IF NOT EXISTS idx_orders_track ON orders(track_number);
    `)
    return err
}

func (s *Store) UpsertOrder(ctx context.Context, orderUID string, track string, raw map[string]interface{}) error {
    b, err := json.Marshal(raw)
    if err != nil { return err }
    query := `
    INSERT INTO orders(order_uid, track_number, data)
    VALUES($1, $2, $3)
    ON CONFLICT (order_uid) DO UPDATE SET
      track_number = EXCLUDED.track_number,
      data = EXCLUDED.data
    `
    _, err = s.DB.ExecContext(ctx, query, orderUID, track, b)
    return err
}

func (s *Store) LoadAllOrders(ctx context.Context) (map[string]json.RawMessage, error) {
    rows, err := s.DB.QueryContext(ctx, `SELECT order_uid, data FROM orders`)
    if err != nil { return nil, err }
    defer rows.Close()
    res := make(map[string]json.RawMessage)
    for rows.Next() {
        var uid string
        var raw json.RawMessage
        if err := rows.Scan(&uid, &raw); err != nil { return nil, err }
        res[uid] = raw
    }
    return res, nil
}

func (s *Store) GetOrder(ctx context.Context, uid string) (json.RawMessage, error) {
    var raw json.RawMessage
    err := s.DB.QueryRowContext(ctx, `SELECT data FROM orders WHERE order_uid=$1`, uid).Scan(&raw)
    return raw, err
}
