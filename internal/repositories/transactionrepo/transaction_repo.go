package transactionrepo

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain"
)

type ITransactionRepository interface {
	Create(ctx context.Context, tx domain.Transaction) error
	GetByHash(ctx context.Context, chainID, txHash string) (domain.Transaction, error)
	GetByID(ctx context.Context, id string) (domain.Transaction, error)
	Update(ctx context.Context, tx domain.Transaction) error
	UpdateStatus(ctx context.Context, id string, status domain.VerificationStatus, metadata map[string]interface{}) error
	GetByAddress(ctx context.Context, chainID, address string, limit, offset int) ([]domain.Transaction, error)
	GetPendingTransactions(ctx context.Context, limit int) ([]domain.Transaction, error)
	GetTransactionsByStatus(ctx context.Context, status domain.VerificationStatus, limit, offset int) ([]domain.Transaction, error)
}
