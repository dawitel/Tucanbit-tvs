package authservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/tuncanbit/tvs/internal/domain"
)

type IAuthService interface {
	VerifyToken(ctx context.Context, tokenString string) (*domain.Claim, error)
	GenerateJWTWithVerification(ctx context.Context, userID uuid.UUID, isVerified, emailVerified, phoneVerified bool) (string, error)
	VerifyAPIKey(ctx context.Context, apiKey string) error
	SaveUserSession(ctx context.Context, session *domain.UserSession) error
}
