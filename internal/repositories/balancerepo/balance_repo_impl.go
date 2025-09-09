package balancerepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/repositories/balancerepo/gen"
)

type BalanceRepository struct {
	db    *sql.DB
	store *gen.Queries
}

func New(db *sql.DB) IBalanceRepository {
	return &BalanceRepository{
		db:    db,
		store: gen.New(db),
	}
}

func (r *BalanceRepository) GetUserBalances(ctx context.Context, userID string) ([]*domain.Balance, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %v", err)
	}

	balances, err := r.store.GetUserBalances(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query balances: %v", err)
	}

	result := make([]*domain.Balance, len(balances))
	for i, b := range balances {
		reservedUnits := "0"
		if b.ReservedUnits.Valid {
			reservedUnits = b.ReservedUnits.String
		}
		amountCents := int64(0)
		if b.AmountCents.Valid {
			amountCents = b.AmountCents.Int64
		}
		reservedCents := int64(0)
		if b.ReservedCents.Valid {
			reservedCents = b.ReservedCents.Int64
		}

		result[i] = &domain.Balance{
			ID:            b.ID.String(),
			UserID:        b.UserID.String(),
			CurrencyCode:  b.CurrencyCode,
			AmountCents:   amountCents,
			AmountUnits:   b.AmountUnits.String,
			ReservedCents: reservedCents,
			ReservedUnits: reservedUnits,
			UpdatedAt:     b.UpdatedAt.Time,
		}
	}

	return result, nil
}

func (r *BalanceRepository) GetBalance(ctx context.Context, userID string) (*domain.Balance, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %v", err)
	}

	balance, err := r.store.GetBalance(ctx, gen.GetBalanceParams{
		UserID:       userUUID,
		CurrencyCode: "USD",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %v", err)
	}

	reservedUnits := "0"
	if balance.ReservedUnits.Valid {
		reservedUnits = balance.ReservedUnits.String
	}
	amountCents := int64(0)
	if balance.AmountCents.Valid {
		amountCents = balance.AmountCents.Int64
	}
	reservedCents := int64(0)
	if balance.ReservedCents.Valid {
		reservedCents = balance.ReservedCents.Int64
	}

	return &domain.Balance{
		ID:            balance.ID.String(),
		UserID:        balance.UserID.String(),
		CurrencyCode:  "USD",
		AmountCents:   amountCents,
		AmountUnits:   balance.AmountUnits.String,
		ReservedCents: reservedCents,
		ReservedUnits: reservedUnits,
		UpdatedAt:     balance.UpdatedAt.Time,
	}, nil
}

func (r *BalanceRepository) ReserveBalance(ctx context.Context, userID, currencyCode string, amountCents int64) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id format: %v", err)
	}

	err = r.store.ReserveBalance(ctx, gen.ReserveBalanceParams{
		UserID:        userUUID,
		CurrencyCode:  currencyCode,
		ReservedCents: sql.NullInt64{Int64: amountCents, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to reserve balance: %v", err)
	}
	return nil
}

func (r *BalanceRepository) UpdateBalance(ctx context.Context, userID, currencyCode string, amountCents int64, amountUnits string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id format: %v", err)
	}

	err = r.store.UpdateBalance(ctx, gen.UpdateBalanceParams{
		UserID:       userUUID,
		CurrencyCode: currencyCode,
		AmountCents:  sql.NullInt64{Int64: amountCents, Valid: true},
		AmountUnits:  sql.NullString{String: amountUnits, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update balance: %v", err)
	}
	return nil
}

func (r *BalanceRepository) ReleaseReservedBalance(ctx context.Context, userID, currencyCode string, amountCents int64) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user_id format: %v", err)
	}

	err = r.store.ReleaseReservedBalance(ctx, gen.ReleaseReservedBalanceParams{
		UserID:        userUUID,
		CurrencyCode:  currencyCode,
		ReservedCents: sql.NullInt64{Int64: amountCents, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to release reserved balance: %v", err)
	}
	return nil
}

func (r *BalanceRepository) LogBalanceChange(ctx context.Context, balanceLog *domain.BalanceLog) error {
	var balanceAfterCents sql.NullInt64
	var balanceAfterUnits, description, transactionID, status sql.NullString
	var timestamp sql.NullTime

	if balanceLog.BalanceAfterCents != 0 {
		balanceAfterCents = sql.NullInt64{Int64: balanceLog.BalanceAfterCents, Valid: true}
	}

	if balanceLog.BalanceAfterUnits != 0 {
		balanceAfterUnits = sql.NullString{String: fmt.Sprintf("%.18f", balanceLog.BalanceAfterUnits), Valid: true}
	}

	if balanceLog.Description != "" {
		description = sql.NullString{String: balanceLog.Description, Valid: true}
	}
	if balanceLog.TransactionID != "" {
		transactionID = sql.NullString{String: balanceLog.TransactionID, Valid: true}
	}
	if balanceLog.Status != "" {
		status = sql.NullString{String: balanceLog.Status, Valid: true}
	}
	if !balanceLog.Timestamp.IsZero() {
		timestamp = sql.NullTime{Time: balanceLog.Timestamp, Valid: true}
	}

	err := r.store.LogBalanceChange(ctx, gen.LogBalanceChangeParams{
		ID:                uuid.MustParse(balanceLog.ID),
		UserID:            uuid.MustParse(balanceLog.UserID),
		Component:         gen.Components(balanceLog.Component),
		CurrencyCode:      balanceLog.CurrencyCode,
		ChangeCents:       balanceLog.ChangeCents,
		ChangeUnits:       fmt.Sprintf("%.18f", balanceLog.ChangeUnits),
		Description:       description,
		Timestamp:         timestamp,
		BalanceAfterCents: balanceAfterCents,
		BalanceAfterUnits: balanceAfterUnits,
		TransactionID:     transactionID,
		Status:            status,
	})
	if err != nil {
		return fmt.Errorf("failed to log balance change: %v", err)
	}
	return nil
}
