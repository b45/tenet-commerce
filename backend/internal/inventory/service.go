package inventory

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// PostMovementTx posts a stock movement within caller's existing transaction.
// Caller owns the transaction lifecycle (commit/rollback).
func (s *Service) PostMovementTx(ctx context.Context, tx pgx.Tx, p PostMovementParams) (*StockMovement, error) {
	return s.repo.PostMovementTx(ctx, tx, p)
}

// GetStockMovements returns paginated movements for a product
func (s *Service) GetStockMovements(ctx context.Context, conn *pgxpool.Conn, productID uuid.UUID, limit, offset int) ([]StockMovement, error) {
	return s.repo.GetStockMovementsByProduct(ctx, conn, productID, limit, offset)
}

// GetReconciledBalance returns the reconciled stock balance verifying:
// opening + sum(subsequent delta) == on_hand
func (s *Service) GetReconciledBalance(ctx context.Context, conn *pgxpool.Conn, productID uuid.UUID) (*CurrentBalance, error) {
	return s.repo.GetReconciledBalance(ctx, conn, productID)
}
