package withdrawalrepo

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain"
)

type IWithdrawalRepository interface {
	UpdateWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error
	LoadPendingWithdrawals(ctx context.Context, limit, offset int) ([]domain.Withdrawal, error)
}
