package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
	auth_repository "github.com/tuncanbit/tvs/internal/repositories/authrepo"
	"github.com/tuncanbit/tvs/pkg/config"
)

type AuthService struct {
	config   *config.Config
	logger   zerolog.Logger
	authRepo auth_repository.IAuthRepository
}

func NewAuthService(config *config.Config, logger zerolog.Logger, authRepo auth_repository.IAuthRepository) *AuthService {
	return &AuthService{
		config:   config,
		logger:   logger,
		authRepo: authRepo,
	}
}

func (s *AuthService) VerifyToken(ctx context.Context, tokenString string) (*domain.Claim, error) {
	jwtSecret := s.config.JWT.Secret
	if jwtSecret == "" {
		s.logger.Error().Msg("JWT secret not configured")
		return nil, fmt.Errorf("JWT secret not configured")
	}

	token, err := jwt.ParseWithClaims(tokenString, &domain.Claim{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		s.logger.Error().Err(err).Str("token", tokenString).Msg("Failed to parse token")
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		s.logger.Error().Str("token", tokenString).Msg("Invalid token")
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*domain.Claim)
	if !ok {
		s.logger.Error().Msg("Invalid claims format")
		return nil, fmt.Errorf("invalid claims format")
	}

	if claims.ExpiresAt < time.Now().Unix() {
		s.logger.Error().Str("token", tokenString).Msg("Token expired")
		return nil, fmt.Errorf("token expired")
	}

	if claims.Issuer != "tucanbit" {
		s.logger.Error().Str("token", tokenString).Msg("Invalid issuer")
		return nil, fmt.Errorf("invalid issuer")
	}

	// session, err := s.authRepo.GetUserSessionByToken(ctx, tokenString)
	// if err != nil {
	// 	s.logger.Error().Err(err).Str("token", tokenString).Msg("Failed to get user session")
	// 	return nil, fmt.Errorf("failed to get user session: %v", err)
	// }
	// if session.ExpiresAt.Before(time.Now()) {
	// 	s.logger.Error().Str("token", tokenString).Msg("Session expired")
	// 	return nil, fmt.Errorf("session expired")
	// }

	return claims, nil
}

func (s *AuthService) GenerateJWTWithVerification(ctx context.Context, userID uuid.UUID, isVerified, emailVerified, phoneVerified bool) (string, error) {
	jwtSecret := s.config.JWT.Secret
	if jwtSecret == "" {
		s.logger.Error().Msg("JWT secret not configured")
		return "", fmt.Errorf("JWT secret not configured")
	}

	expirationTime := time.Now().Add(time.Hour * 24)
	claim := &domain.Claim{
		UserID:        userID,
		IsVerified:    isVerified,
		EmailVerified: emailVerified,
		PhoneVerified: phoneVerified,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "tucanbit",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to sign token")
		return "", fmt.Errorf("failed to sign token: %v", err)
	}

	session := &domain.UserSession{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     tokenString,
		ExpiresAt: expirationTime,
		CreatedAt: time.Now(),
	}
	err = s.authRepo.SaveUserSession(ctx, session)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", userID.String()).Msg("Failed to save user session")
		return "", fmt.Errorf("failed to save user session: %v", err)
	}

	return tokenString, nil
}

func (s *AuthService) SaveUserSession(ctx context.Context, session *domain.UserSession) error {
	err := s.authRepo.SaveUserSession(ctx, session)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", session.UserID.String()).Msg("Failed to save user session")
		return fmt.Errorf("failed to save user session: %v", err)
	}
	return nil
}

func (s *AuthService) VerifyAPIKey(ctx context.Context, apiKey string) error {
	if apiKey == "" {
		return errors.New("invalid API key")
	}

	// apiKeyRecord, err := s.authRepo.GetAPIKey(ctx, apiKey)
	// if err != nil {
	// 	s.logger.Error().Err(err).Str("api_key", apiKey).Msg("Failed to get API key")
	// 	return fmt.Errorf("failed to get API key: %v", err)
	// }

	// if apiKeyRecord == nil {
	// 	return errors.New("invalid API key")
	// }

	return nil
}
