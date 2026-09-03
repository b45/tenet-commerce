package tenant

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxRunner defines the interface for running transactions in a tenant-scoped context.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error
}

// ScopedDB wraps a pooled database connection and enforces transaction-local
// search_path isolation via SET LOCAL.
type ScopedDB struct {
	conn       *pgxpool.Conn
	tenant     *Tenant
	schemaName string
}

// NewScopedDB creates a new ScopedDB wrapping a connection and verified tenant.
func NewScopedDB(conn *pgxpool.Conn, t *Tenant) (*ScopedDB, error) {
	if conn == nil {
		return nil, fmt.Errorf("connection cannot be nil")
	}
	if t == nil {
		return nil, fmt.Errorf("tenant cannot be nil")
	}
	if !IsValidSchemaName(t.SchemaName) {
		return nil, fmt.Errorf("invalid tenant schema name: %s", t.SchemaName)
	}

	return &ScopedDB{
		conn:       conn,
		tenant:     t,
		schemaName: t.SchemaName,
	}, nil
}

// BeginTx begins a database transaction and executes SET LOCAL search_path to the tenant's schema.
// When the transaction commits or rolls back, PostgreSQL automatically clears this local search_path.
func (s *ScopedDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	tx, err := s.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	sanitized := pgx.Identifier{s.schemaName}.Sanitize()
	query := fmt.Sprintf("SET LOCAL search_path TO %s, public;", sanitized)
	if _, err := tx.Exec(ctx, query); err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("set local search_path to %s: %w", s.schemaName, err)
	}

	return tx, nil
}

// RunInTx runs the given function inside a transaction with SET LOCAL search_path.
// If fn returns an error or panics, the transaction is safely rolled back.
func (s *ScopedDB) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := s.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// Conn returns the underlying connection.
func (s *ScopedDB) Conn() *pgxpool.Conn {
	return s.conn
}

// Tenant returns the tenant information.
func (s *ScopedDB) Tenant() *Tenant {
	return s.tenant
}

// SchemaName returns the tenant's trusted schema name.
func (s *ScopedDB) SchemaName() string {
	return s.schemaName
}

// GetScopedDB retrieves the tenant-scoped database wrapper from the Gin context.
func GetScopedDB(c *gin.Context) (*ScopedDB, bool) {
	val, exists := c.Get("tenant_db")
	if !exists {
		return nil, false
	}
	sdb, ok := val.(*ScopedDB)
	return sdb, ok
}

// ExecuteTx is a standalone helper that executes a function inside a transaction
// with SET LOCAL search_path for the given schema.
func ExecuteTx(ctx context.Context, conn *pgxpool.Conn, schemaName string, fn func(tx pgx.Tx) error) (err error) {
	if !IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name: %s", schemaName)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	sanitized := pgx.Identifier{schemaName}.Sanitize()
	query := fmt.Sprintf("SET LOCAL search_path TO %s, public;", sanitized)
	if _, err := tx.Exec(ctx, query); err != nil {
		return fmt.Errorf("set local search_path to %s: %w", schemaName, err)
	}

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
