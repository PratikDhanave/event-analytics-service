package store

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// schemaSQL is embedded so the service can self-bootstrap its database schema.
//
//go:embed schema.sql
var schemaSQL string

// PostgresStore is the durable persistence layer for events.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a connection pool and fails fast if DB is unreachable.
func NewPostgresStore(dbURL string) (*PostgresStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

// EnsureSchema applies schema.sql. Safe to run multiple times.
func (p *PostgresStore) EnsureSchema() error {
	_, err := p.pool.Exec(context.Background(), schemaSQL)
	return err
}

// Ping is used by readiness endpoint to validate DB connectivity.
func (p *PostgresStore) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close shuts down the connection pool.
func (p *PostgresStore) Close() {
	p.pool.Close()
}

// InsertEvent persists an event and returns inserted=false when it is a duplicate.
//
// Duplicate detection is enforced by the database constraint on (tenant_id, event_id),
// which is compatible with retries and at-least-once delivery.
func (p *PostgresStore) InsertEvent(
	ctx context.Context,
	tenantID string,
	eventID string,
	eventName string,
	ts time.Time,
	properties map[string]interface{},
) (bool, error) {

	if tenantID == "" || eventID == "" || eventName == "" {
		return false, errors.New("tenantID/eventID/eventName required")
	}

	if properties == nil {
		properties = map[string]interface{}{}
	}

	propsJSON, err := json.Marshal(properties)
	if err != nil {
		return false, err
	}

	// RETURNING 1 only when inserted; duplicates return no rows.
	var one int
	err = p.pool.QueryRow(ctx, `
		INSERT INTO events(tenant_id, event_id, event_name, ts, properties)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (tenant_id, event_id) DO NOTHING
		RETURNING 1
	`, tenantID, eventID, eventName, ts, propsJSON).Scan(&one)

	if err == nil {
		return true, nil
	}

	// Conflict produces "no rows in result set" because RETURNING returns nothing.
	if err.Error() == "no rows in result set" {
		return false, nil
	}

	return false, err
}

// CountEvents returns the number of events for (tenantID, eventName) in the time window [from,to).
// Using a half-open interval avoids double counting at window boundaries.
func (p *PostgresStore) CountEvents(
	ctx context.Context,
	tenantID string,
	eventName string,
	from time.Time,
	to time.Time,
) (int64, error) {

	var count int64
	err := p.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM events
		WHERE tenant_id=$1
		  AND event_name=$2
		  AND ts >= $3
		  AND ts <  $4
	`, tenantID, eventName, from, to).Scan(&count)

	return count, err
}
