package transactionrepository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/sqlc-dev/pqtype"

	"github.com/tuncanbit/tvs/internal/domain/models"
	"github.com/tuncanbit/tvs/internal/infrastructure/database"
	transactionRepo "github.com/tuncanbit/tvs/internal/repositories/transaction_repo/gen"
)

type transactionRepository struct {
	db     *sql.DB
	store  *transactionRepo.Queries
	logger zerolog.Logger
}

func New(db *database.DBManager, logger zerolog.Logger) ITransactionRepository {
	return &transactionRepository{
		db:     db.Db,
		store:  transactionRepo.New(db.Db),
		logger: logger,
	}
}

func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	params := transactionRepo.CreateTransactionParams{
		ChainID:       tx.ChainID,
		ChainType:     string(tx.ChainType),
		TxHash:        tx.TxHash,
		FromAddress:   tx.FromAddress,
		ToAddress:     tx.ToAddress,
		Amount:        tx.Amount,
		Fee:           sql.NullString{String: tx.Fee, Valid: true},
		BlockNumber:   sql.NullInt64{Int64: *tx.BlockNumber, Valid: true},
		BlockHash:     sql.NullString{String: *tx.BlockHash, Valid: true},
		Status:        string(tx.Status),
		Confirmations: int32(tx.Confirmations),
		Timestamp:     tx.Timestamp,
		VerifiedAt:    sql.NullTime{Time: *tx.VerifiedAt, Valid: tx.VerifiedAt != nil},
		ProcessorID:   sql.NullString{String: *tx.ProcessorID, Valid: true},
		Metadata:      pqtype.NullRawMessage{RawMessage: tx.Metadata, Valid: tx.Metadata != nil},
	}

	err := r.store.CreateTransaction(ctx, params)

	if err != nil {
		r.logger.Error().Err(err).Str("tx_hash", tx.TxHash).Msg("Failed to create transaction")
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) GetByHash(ctx context.Context, chainID, txHash string) (*models.Transaction, error) {
	params := transactionRepo.GetTransactionByHashParams{
		ChainID: chainID,
		TxHash:  txHash,
	}

	txRow, err := r.store.GetTransactionByHash(ctx, params)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		r.logger.Error().Err(err).Str("chain_id", chainID).Str("tx_hash", txHash).Msg("Failed to get transaction by hash")
		return nil, fmt.Errorf("failed to get transaction by hash: %w", err)
	}

	return &models.Transaction{
		ID:            txRow.ID.String(),
		ChainID:       txRow.ChainID,
		ChainType:     models.ChainType(txRow.ChainType),
		TxHash:        txRow.TxHash,
		FromAddress:   txRow.FromAddress,
		ToAddress:     txRow.ToAddress,
		Amount:        txRow.Amount,
		Fee:           txRow.Fee.String,
		BlockNumber:   &txRow.BlockNumber.Int64,
		BlockHash:     &txRow.BlockHash.String,
		Status:        models.VerificationStatus(txRow.Status),
		Confirmations: int(txRow.Confirmations),
		Timestamp:     txRow.Timestamp,
		VerifiedAt:    &txRow.VerifiedAt.Time,
		ProcessorID:   &txRow.ProcessorID.String,
		Metadata:      txRow.Metadata.RawMessage,
		CreatedAt:     txRow.CreatedAt,
		UpdatedAt:     txRow.UpdatedAt,
	}, nil
}

func (r *transactionRepository) GetByID(ctx context.Context, id string) (*models.Transaction, error) {
	txRow, err := r.store.GetTransactionByID(ctx, uuid.MustParse(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		r.logger.Error().Err(err).Str("id", id).Msg("Failed to get transaction by ID")
		return nil, fmt.Errorf("failed to get transaction by ID: %w", err)
	}

	return &models.Transaction{
		ID:            txRow.ID.String(),
		ChainID:       txRow.ChainID,
		ChainType:     models.ChainType(txRow.ChainType),
		TxHash:        txRow.TxHash,
		FromAddress:   txRow.FromAddress,
		ToAddress:     txRow.ToAddress,
		Amount:        txRow.Amount,
		Fee:           txRow.Fee.String,
		BlockNumber:   &txRow.BlockNumber.Int64,
		BlockHash:     &txRow.BlockHash.String,
		Status:        models.VerificationStatus(txRow.Status),
		Confirmations: int(txRow.Confirmations),
		Timestamp:     txRow.Timestamp,
		VerifiedAt:    &txRow.VerifiedAt.Time,
		ProcessorID:   &txRow.ProcessorID.String,
		Metadata:      txRow.Metadata.RawMessage,
		CreatedAt:     txRow.CreatedAt,
		UpdatedAt:     txRow.UpdatedAt,
	}, nil
}

func (r *transactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	params := transactionRepo.UpdateTransactionParams{
		ID:            uuid.MustParse(tx.ID),
		ChainID:       tx.ChainID,
		ChainType:     string(tx.ChainType),
		TxHash:        tx.TxHash,
		FromAddress:   tx.FromAddress,
		ToAddress:     tx.ToAddress,
		Amount:        tx.Amount,
		Fee:           sql.NullString{String: tx.Fee, Valid: true},
		BlockNumber:   sql.NullInt64{Int64: *tx.BlockNumber, Valid: true},
		BlockHash:     sql.NullString{String: *tx.BlockHash, Valid: true},
		Status:        string(tx.Status),
		Confirmations: int32(tx.Confirmations),
		Timestamp:     tx.Timestamp,
		VerifiedAt:    sql.NullTime{Time: *tx.VerifiedAt, Valid: tx.VerifiedAt != nil},
		ProcessorID:   sql.NullString{String: *tx.ProcessorID, Valid: true},
		Metadata:      pqtype.NullRawMessage{RawMessage: tx.Metadata, Valid: tx.Metadata != nil},
	}

	_, err := r.store.UpdateTransaction(ctx, params)
	if err != nil {
		r.logger.Error().Err(err).Str("id", tx.ID).Msg("Failed to update transaction")
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) UpdateStatus(ctx context.Context, id string, status models.VerificationStatus, metadata map[string]interface{}) error {
	var metadataJSON []byte
	var err error

	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	var verifiedAt *time.Time
	if status == models.StatusVerified {
		now := time.Now()
		verifiedAt = &now
	}
	params := transactionRepo.UpdateTransactionStatusParams{
		ID:         uuid.MustParse(id),
		Status:     string(status),
		VerifiedAt: sql.NullTime{Time: *verifiedAt, Valid: verifiedAt != nil},
		Metadata:   pqtype.NullRawMessage{RawMessage: metadataJSON, Valid: metadataJSON != nil},
	}

	_, err = r.store.UpdateTransactionStatus(ctx, params)
	if err != nil {
		r.logger.Error().Err(err).Str("id", id).Str("status", string(status)).Msg("Failed to update transaction status")
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

func (r *transactionRepository) GetByAddress(ctx context.Context, chainID, address string, limit, offset int) ([]*models.Transaction, error) {
	params := transactionRepo.GetTransactionsByAddressParams{
		ChainID:     chainID,
		FromAddress: address,
		Limit:       int32(limit),
		Offset:      int32(offset),
	}

	rows, err := r.store.GetTransactionsByAddress(ctx, params)
	if err != nil {
		r.logger.Error().Err(err).Str("chain_id", chainID).Str("address", address).Msg("Failed to get transactions by address")
		return nil, fmt.Errorf("failed to get transactions by address: %w", err)
	}

	var transactions []*models.Transaction
	for _, txRow := range rows {
		transactions = append(transactions, &models.Transaction{
			ID:            txRow.ID.String(),
			ChainID:       txRow.ChainID,
			ChainType:     models.ChainType(txRow.ChainType),
			TxHash:        txRow.TxHash,
			FromAddress:   txRow.FromAddress,
			ToAddress:     txRow.ToAddress,
			Amount:        txRow.Amount,
			Fee:           txRow.Fee.String,
			BlockNumber:   &txRow.BlockNumber.Int64,
			BlockHash:     &txRow.BlockHash.String,
			Status:        models.VerificationStatus(txRow.Status),
			Confirmations: int(txRow.Confirmations),
			Timestamp:     txRow.Timestamp,
			VerifiedAt:    &txRow.VerifiedAt.Time,
			ProcessorID:   &txRow.ProcessorID.String,
			Metadata:      txRow.Metadata.RawMessage,
			CreatedAt:     txRow.CreatedAt,
			UpdatedAt:     txRow.UpdatedAt,
		})
	}

	return transactions, nil
}

// GetPendingTransactions retrieves pending transactions for processing
func (r *transactionRepository) GetPendingTransactions(ctx context.Context, limit int) ([]*models.Transaction, error) {
	rows, err := r.store.GetPendingTransactions(ctx, int32(limit))
	if err != nil {
		r.logger.Error().Err(err).Msg("Failed to get pending transactions")
		return nil, fmt.Errorf("failed to get pending transactions: %w", err)
	}

	var transactions []*models.Transaction
	for _, txRow := range rows {
		transactions = append(transactions, &models.Transaction{
			ID:            txRow.ID.String(),
			ChainID:       txRow.ChainID,
			ChainType:     models.ChainType(txRow.ChainType),
			TxHash:        txRow.TxHash,
			FromAddress:   txRow.FromAddress,
			ToAddress:     txRow.ToAddress,
			Amount:        txRow.Amount,
			Fee:           txRow.Fee.String,
			BlockNumber:   &txRow.BlockNumber.Int64,
			BlockHash:     &txRow.BlockHash.String,
			Status:        models.VerificationStatus(txRow.Status),
			Confirmations: int(txRow.Confirmations),
			Timestamp:     txRow.Timestamp,
			VerifiedAt:    &txRow.VerifiedAt.Time,
			ProcessorID:   &txRow.ProcessorID.String,
			Metadata:      txRow.Metadata.RawMessage,
			CreatedAt:     txRow.CreatedAt,
			UpdatedAt:     txRow.UpdatedAt,
		})
	}

	return transactions, nil
}

func (r *transactionRepository) GetTransactionsByStatus(ctx context.Context, status models.VerificationStatus, limit, offset int) ([]*models.Transaction, error) {
	params := transactionRepo.GetTransactionsByStatusParams{
		Status: string(status),
		Limit:  int32(limit),
		Offset: int32(offset),
	}
	rows, err := r.store.GetTransactionsByStatus(ctx, params)
	if err != nil {
		r.logger.Error().Err(err).Str("status", string(status)).Msg("Failed to get transactions by status")
		return nil, fmt.Errorf("failed to get transactions by status: %w", err)
	}

	var transactions []*models.Transaction
	for _, txRow := range rows {
		transactions = append(transactions, &models.Transaction{
			ID:            txRow.ID.String(),
			ChainID:       txRow.ChainID,
			ChainType:     models.ChainType(txRow.ChainType),
			TxHash:        txRow.TxHash,
			FromAddress:   txRow.FromAddress,
			ToAddress:     txRow.ToAddress,
			Amount:        txRow.Amount,
			Fee:           txRow.Fee.String,
			BlockNumber:   &txRow.BlockNumber.Int64,
			BlockHash:     &txRow.BlockHash.String,
			Status:        models.VerificationStatus(txRow.Status),
			Confirmations: int(txRow.Confirmations),
			Timestamp:     txRow.Timestamp,
			VerifiedAt:    &txRow.VerifiedAt.Time,
			ProcessorID:   &txRow.ProcessorID.String,
			Metadata:      txRow.Metadata.RawMessage,
			CreatedAt:     txRow.CreatedAt,
			UpdatedAt:     txRow.UpdatedAt,
		})
	}

	return transactions, nil
}

func (r *transactionRepository) Delete(ctx context.Context, id string) error {
	_, err := r.store.DeleteTransaction(ctx, uuid.MustParse(id))
	if err != nil {
		r.logger.Error().Err(err).Str("id", id).Msg("Failed to delete transaction")
		return fmt.Errorf("failed to delete transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) GetStats(ctx context.Context, chainID string) (map[string]interface{}, error) {
	stats, err := r.store.GetTransactionStats(ctx, chainID)
	if err != nil {
		r.logger.Error().Err(err).Str("chain_id", chainID).Msg("Failed to get transaction stats")
		return nil, fmt.Errorf("failed to get transaction stats: %w", err)
	}

	return map[string]interface{}{
		"chain_id":           chainID,
		"total_transactions": stats.TotalTransactions,
		"verified_count":     stats.VerifiedCount,
		"pending_count":      stats.PendingCount,
		"failed_count":       stats.FailedCount,
		"processing_count":   stats.ProcessingCount,
		"total_amount":       stats.TotalAmount,
		"avg_confirmations":  stats.AvgConfirmations,
	}, nil
}
