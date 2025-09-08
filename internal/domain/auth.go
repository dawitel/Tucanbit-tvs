package domain

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	KycStatus   string    `json:"kyc_status"`
	Email       string    `json:"email"`
	PhoneNumber string    `json:"phone_number"`
	UserType    string    `json:"user_type"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserSession struct {
	ID                    uuid.UUID `json:"id"`
	UserID                uuid.UUID `json:"user_id"`
	Token                 string    `json:"token"`
	ExpiresAt             time.Time `json:"expires_at"`
	IPAddress             string    `json:"ip_address,omitempty"`
	UserAgent             string    `json:"user_agent,omitempty"`
	RefreshToken          string    `json:"refresh_token,omitempty"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at,omitempty"`
	CreatedAt             time.Time `json:"created_at"`
}

type Claim struct {
	UserID        uuid.UUID `json:"user_id"`
	IsVerified    bool      `json:"is_verified"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	jwt.StandardClaims
}
