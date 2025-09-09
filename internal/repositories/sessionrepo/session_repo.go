package sessionrepo

import (
	"context"
	"database/sql"

	"github.com/tuncanbit/tvs/internal/domain"
)

type ISessionRepository interface {
	LoadPendingDepositSessions(ctx context.Context, limit, offset int) ([]domain.DepositSession, error)
	GetBySessionIDTx(ctx context.Context, tx *sql.Tx, sessionID string) (domain.DepositSession, error)
	UpdateDepositSessionStatusTx(ctx context.Context, tx *sql.Tx, sessionId string, status string) error
	UpdateDepositSessionStatus(ctx context.Context, sessionId string, status string, errorMessage string) error
	CompleteSession(ctx context.Context, session domain.DepositSession, errorMessage string) error
	BeginTx(ctx context.Context) (*sql.Tx, error)
}
