package withdrawalrepo

import (
	"context"
	"database/sql"

	"github.com/tuncanbit/tvs/internal/domain"
)

type IWithdrawalRepository interface {
	UpdateWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
	GetByWithdrawalIDTx(ctx context.Context, tx *sql.Tx, withdrawalId string) (*domain.Withdrawal, error)
	UpdateWithdrawalStatusTx(ctx context.Context, tx *sql.Tx, withdrawalId string, status domain.WithdrawalStatus) error
	UpdateWithdrawalStatus(ctx context.Context, withdrawalId string, status domain.WithdrawalStatus, errorMessage string) error
	LoadPendingWithdrawals(ctx context.Context, limit, offset int) ([]domain.Withdrawal, error)
}
