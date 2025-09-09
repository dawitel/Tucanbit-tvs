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

func (r *WithdrawalRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}
func (r *WithdrawalRepository) GetByWithdrawalIDTx(ctx context.Context, tx *sql.Tx, withdrawalId string) (*domain.Withdrawal, error) {
	txStore := r.queries.WithTx(tx)
	dbWithdrawal, err := txStore.GetWithdrawalByID(ctx, withdrawalId)
	if err != nil {
		return nil, err
	}

	withdrawal := mapDBWithdrawalToDomain(dbWithdrawal)
	return withdrawal, nil
}

func mapDBWithdrawalToDomain(dbWithdrawal gen.Withdrawal) *domain.Withdrawal {
	return &domain.Withdrawal{
		ID:                    dbWithdrawal.ID.String(),
		UserID:                dbWithdrawal.UserID.String(),
		AdminID:               dbWithdrawal.AdminID.UUID.String(),
		WithdrawalID:          dbWithdrawal.WithdrawalID,
		ChainID:               dbWithdrawal.ChainID,
		Network:               dbWithdrawal.Network,
		CryptoCurrency:        dbWithdrawal.CryptoCurrency,
		USDAmountCents:        dbWithdrawal.UsdAmountCents,
		CryptoAmount:          dbWithdrawal.CryptoAmount,
		ExchangeRate:          dbWithdrawal.ExchangeRate,
		FeeCents:              dbWithdrawal.FeeCents,
		ToAddress:             dbWithdrawal.ToAddress,
		TxHash:                dbWithdrawal.TxHash.String,
		Status:                domain.WithdrawalStatus(dbWithdrawal.Status),
		RequiresAdminReview:   dbWithdrawal.RequiresAdminReview,
		AdminReviewDeadline:   dbWithdrawal.AdminReviewDeadline.Time,
		ProcessedBySystem:     dbWithdrawal.ProcessedBySystem.Bool,
		SourceWalletAddress:   dbWithdrawal.SourceWalletAddress,
		AmountReservedCents:   dbWithdrawal.AmountReservedCents,
		ReservationReleased:   dbWithdrawal.ReservationReleased.Bool,
		ReservationReleasedAt: dbWithdrawal.ReservationReleasedAt.Time,
		CreatedAt:             dbWithdrawal.CreatedAt.Time,
		UpdatedAt:             dbWithdrawal.UpdatedAt.Time,
	}
}

func (r *WithdrawalRepository) UpdateWithdrawalStatusTx(ctx context.Context, tx *sql.Tx, withdrawalId string, status domain.WithdrawalStatus) error {
	txStore := r.queries.WithTx(tx)
	return txStore.UpdateWithdrawal(ctx, gen.UpdateWithdrawalParams{
		WithdrawalID: withdrawalId,
		Status:       gen.WithdrawalStatus(status),
	})
}

func (r *WithdrawalRepository) UpdateWithdrawalStatus(ctx context.Context, withdrawalId string, status domain.WithdrawalStatus, errorMessage string) error {
	return r.queries.UpdateWithdrawal(ctx, gen.UpdateWithdrawalParams{
		WithdrawalID: withdrawalId,
		Status:       gen.WithdrawalStatus(status),
		ErrorMessage: sql.NullString{String: errorMessage, Valid: errorMessage != ""},
	})
}
