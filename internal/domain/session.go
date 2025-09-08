package domain

import (
	"encoding/json"
	"time"
)

type SessionStatus string

const (
	SessionStatusPending    SessionStatus = "pending"
	SessionStatusProcessing SessionStatus = "processing"
	SessionStatusCompleted  SessionStatus = "completed"
	SessionStatusFailed     SessionStatus = "failed"
	SessionStatusCancelled  SessionStatus = "cancelled"
	SessionStatusExpired    SessionStatus = "expired"
)

type DepositSession struct {
	ID             string          `json:"id" db:"id"`
	SessionID      string          `json:"session_id" db:"session_id" binding:"required"`
	UserID         string          `json:"user_id" db:"user_id" binding:"required"`
	ChainID        string          `json:"chain_id" db:"chain_id" binding:"required"`
	Network        string          `json:"network" db:"network" binding:"required"`
	WalletAddress  string          `json:"wallet_address" db:"wallet_address"`
	Amount         float64         `json:"amount" db:"amount" binding:"required"`
	CryptoCurrency string          `json:"crypto_currency" db:"crypto_currency" binding:"required"`
	Status         SessionStatus   `json:"status" db:"status" binding:"required"`
	QRCodeData     string          `json:"qr_code_data" db:"qr_code_data"`
	PaymentLink    string          `json:"payment_link" db:"payment_link"`
	Metadata       json.RawMessage `json:"metadata" db:"metadata"`
	ErrorMessage   string          `json:"error_message" db:"error_message"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at" binding:"required"`
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at" binding:"required"`
}
