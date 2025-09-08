package balancerepo

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain"
)

type IBalanceRepository interface {
	GetUserBalances(ctx context.Context, userID string) ([]*domain.Balance, error)
	GetBalance(ctx context.Context, userID, currencyCode string) (*domain.Balance, error)
	ReserveBalance(ctx context.Context, userID, currencyCode string, amountCents int64) error
	ReleaseReservedBalance(ctx context.Context, userID, currencyCode string, amountCents int64) error
	LogBalanceChange(ctx context.Context, balanceLog *domain.BalanceLog) error
	UpdateBalance(ctx context.Context, userID, currency string, newAmountCents int64, newAmountUnits string) error
}
