package withdrawalrepo

import (
	"context"
	"database/sql"
	"time"

	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/repositories/withdrawalrepo/gen"
)

type WithdrawalRepository struct {
	db      *sql.DB
	queries *gen.Queries
	logger  zerolog.Logger
}

func New(db *sql.DB, logger zerolog.Logger) IWithdrawalRepository {
	return &WithdrawalRepository{
		db:      db,
		queries: gen.New(db),
		logger:  logger,
	}
}

func (r *WithdrawalRepository) UpdateWithdrawal(ctx context.Context, withdrawal domain.Withdrawal) error {
	err := r.queries.UpdateWithdrawal(ctx, gen.UpdateWithdrawalParams{
		TxHash:       sql.NullString{String: withdrawal.TxHash, Valid: withdrawal.TxHash != ""},
		Status:       gen.WithdrawalStatus(withdrawal.Status),
		WithdrawalID: withdrawal.WithdrawalID,
	})
	if err != nil {
		r.logger.Err(err).Str("withdrawal_id", withdrawal.WithdrawalID).Msg("Failed to update withdrawal")
		return err
	}
	return nil
}

func (r *WithdrawalRepository) LoadPendingWithdrawals(ctx context.Context, limit, offset int) ([]domain.Withdrawal, error) {
	resp, err := r.queries.ListPendingWithdrawals(ctx, gen.ListPendingWithdrawalsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		r.logger.Err(err).Msg("Failed to load pending withdrawals")
		return nil, err
	}

	withdrawals := make([]domain.Withdrawal, len(resp))
	for i, w := range resp {
		var adminReviewDeadline time.Time
		if w.AdminReviewDeadline.Valid {
			adminReviewDeadline = w.AdminReviewDeadline.Time
		}

		var reservationReleasedAt time.Time
		if w.ReservationReleasedAt.Valid {
			reservationReleasedAt = w.ReservationReleasedAt.Time
		}

		var adminID string
		if w.AdminID.Valid {
			adminID = w.AdminID.UUID.String()
		}

		var createdAt, updatedAt time.Time
		if w.CreatedAt.Valid {
			createdAt = w.CreatedAt.Time
		}
		if w.UpdatedAt.Valid {
			updatedAt = w.UpdatedAt.Time
		}

		withdrawals[i] = domain.Withdrawal{
			ID:                    w.ID.String(),
			UserID:                w.UserID.String(),
			AdminID:               adminID,
			WithdrawalID:          w.WithdrawalID,
			ChainID:               w.ChainID,
			Network:               w.Network,
			CryptoCurrency:        w.CryptoCurrency,
			USDAmountCents:        w.UsdAmountCents,
			CryptoAmount:          w.CryptoAmount,
			ExchangeRate:          w.ExchangeRate,
			FeeCents:              w.FeeCents,
			ToAddress:             w.ToAddress,
			TxHash:                w.TxHash.String,
			Status:                domain.WithdrawalStatus(w.Status),
			RequiresAdminReview:   w.RequiresAdminReview,
			AdminReviewDeadline:   adminReviewDeadline,
			ProcessedBySystem:     w.ProcessedBySystem.Bool,
			SourceWalletAddress:   w.SourceWalletAddress,
			AmountReservedCents:   w.AmountReservedCents,
			ReservationReleased:   w.ReservationReleased.Bool,
			ReservationReleasedAt: reservationReleasedAt,
			CreatedAt:             createdAt,
			UpdatedAt:             updatedAt,
		}
	}
	return withdrawals, nil
}
