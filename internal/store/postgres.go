package store

import (
	"context"
	_ "embed" // Required for embedding SQL file into binary
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Embed schema SQL so the binary always knows how to initialize DB.
//
//go:embed schema.sql
var schemaSQL string

// PostgresStore wraps a pgx connection pool.
// This is the durable storage layer for events.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a connection pool with startup timeout.
func NewPostgresStore(dbURL string) (*PostgresStore, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}

	// Verify DB reachable immediately.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

// EnsureSchema applies the embedded SQL schema.
// Safe to run multiple times.
func (p *PostgresStore) EnsureSchema() error {
	_, err := p.pool.Exec(context.Background(), schemaSQL)
	return err
}

// Ping checks DB health.
func (p *PostgresStore) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

// Close shuts down connection pool.
func (p *PostgresStore) Close() {
	p.pool.Close()
}
