package sessionrepo

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain"
)

type ISessionRepository interface {
	LoadPendingDepositSessions(ctx context.Context, limit, offset int) ([]domain.DepositSession, error)
	UpdateDepositSessionStatus(ctx context.Context, sessionId string, status string, errorMessage string) error
	CompleteSession(ctx context.Context, session domain.DepositSession, errorMessage string) error
}
