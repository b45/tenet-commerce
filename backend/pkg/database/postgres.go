package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB represents the database connection pool
type PostgresDB struct {
	Pool *pgxpool.Pool
}

// NewPostgresDB initializes and returns a new PostgreSQL connection pool
func NewPostgresDB(ctx context.Context) (*PostgresDB, error) {
	host := os.Getenv("DATABASE_HOST")
	port := os.Getenv("DATABASE_PORT")
	user := os.Getenv("DATABASE_USER")
	password := os.Getenv("DATABASE_PASSWORD")
	dbname := os.Getenv("DATABASE_NAME")
	sslmode := os.Getenv("DATABASE_SSLMODE")

	if port == "" {
		port = "5433"
	}
	if host == "" {
		host = "127.0.0.1"
		user = "postgres"
		password = "postgres"
		dbname = "tenet_commerce"
		sslmode = "disable"
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("error parsing db config: %w", err)
	}

	// Set pool configuration suitable for multi-tenant SaaS
	config.MaxConns = 50
	config.MinConns = 10
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("error pinging db: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL Database Pool")
	return &PostgresDB{Pool: pool}, nil
}

// Close closes the connection pool
func (db *PostgresDB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
		log.Println("PostgreSQL connection pool closed")
	}
}
