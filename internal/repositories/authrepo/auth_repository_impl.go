package authrepo

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/repositories/authrepo/gen"
)

type AuthRepository struct {
	db    *sql.DB
	store *gen.Queries
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{
		db:    db,
		store: gen.New(db),
	}
}

func (r *AuthRepository) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %v", err)
	}

	user, err := r.store.GetUserByID(ctx, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return &domain.User{
		ID:          user.ID,
		Username:    user.Username,
		KycStatus:   user.KycStatus.String,
		Email:       user.Email.String,
		PhoneNumber: user.PhoneNumber,
		UserType:    user.UserType.String,
		Status:      user.Status.String,
		CreatedAt:   user.CreatedAt.Time,
		UpdatedAt:   user.UpdatedAt.Time,
	}, nil
}

func (r *AuthRepository) SaveUserSession(ctx context.Context, session *domain.UserSession) error {
	var refreshToken, userAgent sql.NullString
	var refreshTokenExpiresAt sql.NullTime

	if session.RefreshToken != "" {
		refreshToken = sql.NullString{String: session.RefreshToken, Valid: true}
	}

	if session.UserAgent != "" {
		userAgent = sql.NullString{String: session.UserAgent, Valid: true}
	}
	if !session.RefreshTokenExpiresAt.IsZero() {
		refreshTokenExpiresAt = sql.NullTime{Time: session.RefreshTokenExpiresAt, Valid: true}
	}

	var ipNet pqtype.Inet
	if session.IPAddress != "" {
		ip := net.ParseIP(session.IPAddress)
		if ip != nil {
			if ip.To4() != nil {
				ipNet = pqtype.Inet{
					IPNet: net.IPNet{
						IP:   ip,
						Mask: net.CIDRMask(32, 32),
					},
					Valid: true,
				}
			} else {
				ipNet = pqtype.Inet{
					IPNet: net.IPNet{
						IP:   ip,
						Mask: net.CIDRMask(128, 128),
					},
					Valid: true,
				}
			}
		}
	}

	userIDValid := session.UserID != uuid.Nil && session.UserID.String() != ""
	err := r.store.SaveUserSession(ctx, gen.SaveUserSessionParams{
		ID:                    session.ID,
		UserID:                uuid.NullUUID{UUID: session.UserID, Valid: userIDValid},
		Token:                 session.Token,
		ExpiresAt:             session.ExpiresAt,
		IpAddress:             ipNet,
		UserAgent:             userAgent,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
		CreatedAt:             sql.NullTime{Time: session.CreatedAt, Valid: !session.CreatedAt.IsZero()},
	})
	if err != nil {
		return fmt.Errorf("failed to save user session: %v", err)
	}
	return nil
}

func (r *AuthRepository) GetUserSessionByToken(ctx context.Context, token string) (*domain.UserSession, error) {
	session, err := r.store.GetUserSessionByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get user session: %v", err)
	}

	userSession := &domain.UserSession{
		ID:        session.ID,
		UserID:    session.UserID.UUID,
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt.Time,
	}
	if session.IpAddress.Valid {
		userSession.IPAddress = session.IpAddress.IPNet.String()
	}
	if session.UserAgent.Valid {
		userSession.UserAgent = session.UserAgent.String
	}
	if session.RefreshToken.Valid {
		userSession.RefreshToken = session.RefreshToken.String
	}
	if session.RefreshTokenExpiresAt.Valid {
		userSession.RefreshTokenExpiresAt = session.RefreshTokenExpiresAt.Time
	}

	return userSession, nil
}
