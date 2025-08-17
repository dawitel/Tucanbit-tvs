package transactionrepository

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain/models"
)

type ITransactionRepository interface {
	Create(ctx context.Context, tx *models.Transaction) error
	GetByHash(ctx context.Context, chainID, txHash string) (*models.Transaction, error)
	GetByID(ctx context.Context, id string) (*models.Transaction, error)
	Update(ctx context.Context, tx *models.Transaction) error
	UpdateStatus(ctx context.Context, id string, status models.VerificationStatus, metadata map[string]interface{}) error
	GetByAddress(ctx context.Context, chainID, address string, limit, offset int) ([]*models.Transaction, error)
	GetPendingTransactions(ctx context.Context, limit int) ([]*models.Transaction, error)
	GetTransactionsByStatus(ctx context.Context, status models.VerificationStatus, limit, offset int) ([]*models.Transaction, error)
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context, chainID string) (map[string]interface{}, error)
}
