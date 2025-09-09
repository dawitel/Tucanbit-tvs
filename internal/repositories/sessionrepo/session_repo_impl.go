package sessionrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/sqlc-dev/pqtype"

	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/infrastructure/database"
	sessionRepo "github.com/tuncanbit/tvs/internal/repositories/sessionrepo/gen"
)

type sessionRepositoryImpl struct {
	db     *sql.DB
	store  *sessionRepo.Queries
	logger zerolog.Logger
}

func New(db *database.DBManager, logger zerolog.Logger) ISessionRepository {
	return &sessionRepositoryImpl{
		db:     db.Db,
		store:  sessionRepo.New(db.Db),
		logger: logger,
	}
}

func (r *sessionRepositoryImpl) LoadPendingDepositSessions(ctx context.Context, limit, offset int) ([]domain.DepositSession, error) {
	params := sessionRepo.ListPendingDepositSessionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	}

	sessions, err := r.store.ListPendingDepositSessions(ctx, params)
	if err != nil {
		r.logger.Err(err).
			Msg("Failed to list sessions")
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	result := make([]domain.DepositSession, len(sessions))
	for i, session := range sessions {
		result[i] = r.convertFromDB(&session)
	}
	return result, nil
}

func (r *sessionRepositoryImpl) UpdateDepositSessionStatus(ctx context.Context, sessionId, status string, errorMessage string) error {
	params := sessionRepo.UpdateDepositSessionStatusParams{
		SessionID:    sessionId,
		Status:       sessionRepo.DepositSessionStatus(status),
		ErrorMessage: sql.NullString{String: errorMessage, Valid: errorMessage != ""},
	}

	if err := r.store.UpdateDepositSessionStatus(ctx, params); err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionId).Str("status", string(status)).Msg("Failed to update deposit session status")
		return fmt.Errorf("failed to update desposit session status: %w", err)
	}

	return nil
}

func (r *sessionRepositoryImpl) CompleteSession(ctx context.Context, session domain.DepositSession, errorMessage string) error {
	params := sessionRepo.CompleteDepositSessionParams{
		SessionID:    session.SessionID,
		Status:       sessionRepo.DepositSessionStatus(session.Status),
		Metadata:     pqtype.NullRawMessage{RawMessage: session.Metadata, Valid: session.Metadata != nil},
		ErrorMessage: sql.NullString{String: errorMessage, Valid: errorMessage != ""},
	}

	if err := r.store.CompleteDepositSession(ctx, params); err != nil {
		r.logger.Error().Err(err).Str("session_id", session.SessionID).Str("status", string(session.Status)).Msg("Failed to complete deposit session status")
		return fmt.Errorf("failed to complete deposit session status: %w", err)
	}

	return nil
}

func (r *sessionRepositoryImpl) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *sessionRepositoryImpl) GetBySessionIDTx(ctx context.Context, tx *sql.Tx, sessionID string) (domain.DepositSession, error) {
	txStore := r.store.WithTx(tx)
	session, err := txStore.GetDepositSessionByID(ctx, sessionID)
	if err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionID).Msg("Failed to get deposit session")
		return domain.DepositSession{}, fmt.Errorf("failed to get deposit session: %w", err)
	}

	return r.convertFromDB(&session), nil
}

func (r *sessionRepositoryImpl) UpdateDepositSessionStatusTx(ctx context.Context, tx *sql.Tx, sessionId string, status string) error {
	txStore := r.store.WithTx(tx)
	if err := txStore.UpdateDepositSessionStatus(ctx, sessionRepo.UpdateDepositSessionStatusParams{
		SessionID: sessionId,
		Status:    sessionRepo.DepositSessionStatus(status),
	}); err != nil {
		r.logger.Error().Err(err).Str("session_id", sessionId).Str("status", status).Msg("Failed to update deposit session status")
		return fmt.Errorf("failed to update deposit session status: %w", err)
	}

	return nil
}

func (r *sessionRepositoryImpl) convertFromDB(session *sessionRepo.DepositSession) domain.DepositSession {
	amtFloat := float64(0)
	var err error
	if session.Amount != "" {
		amtFloat, err = strconv.ParseFloat(session.Amount, 64)
		if err != nil {
			r.logger.Err(err).
				Msg("Failed to parse amount")
			return domain.DepositSession{}
		}
	}

	return domain.DepositSession{
		ID:             session.ID.String(),
		SessionID:      session.SessionID,
		UserID:         session.UserID.String(),
		Network:        session.Network,
		ChainID:        session.ChainID,
		WalletAddress:  session.WalletAddress.String,
		Amount:         amtFloat,
		CryptoCurrency: session.CryptoCurrency,
		Status:         domain.SessionStatus(session.Status),
		QRCodeData:     session.QrCodeData.String,
		PaymentLink:    session.PaymentLink.String,
		Metadata:       session.Metadata.RawMessage,
		ErrorMessage:   session.ErrorMessage.String,
		CreatedAt:      session.CreatedAt.Time,
		UpdatedAt:      session.UpdatedAt.Time,
	}
}
