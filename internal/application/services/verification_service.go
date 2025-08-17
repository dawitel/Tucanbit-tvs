package services

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain/models"
)

type IVerificationService interface {
	VerifyTransaction(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error)
	VerifyBatch(ctx context.Context, requests []*models.VerificationRequest) ([]*models.VerificationResponse, error)
	GetTransactionStatus(ctx context.Context, chainID, txHash string) (*models.Transaction, error)
	GetTransactionsByAddress(ctx context.Context, chainID, address string, limit, offset int) ([]*models.Transaction, error)
	ReprocessTransaction(ctx context.Context, id string) (*models.VerificationResponse, error)
}
