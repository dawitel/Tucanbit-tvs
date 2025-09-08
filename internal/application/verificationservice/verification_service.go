package verificationservice

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain"
)

type IVerificationService interface {
	StartTransactionVerification(ctx context.Context) error
	VerifyTransactionFromPDMWebhook(ctx context.Context, req domain.PDMWebhookRequest) error
}
