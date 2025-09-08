package authrepo

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain"
)

type IAuthRepository interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	SaveUserSession(ctx context.Context, session *domain.UserSession) error
	GetUserSessionByToken(ctx context.Context, token string) (*domain.UserSession, error)
}
