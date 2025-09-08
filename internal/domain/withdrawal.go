package domain

import (
	"encoding/json"
	"time"
)

type WithdrawalStatus string

const (
	WithdrawalStatusPending             WithdrawalStatus = "pending"
	WithdrawalStatusProcessing          WithdrawalStatus = "processing"
	WithdrawalStatusCompleted           WithdrawalStatus = "completed"
	WithdrawalStatusFailed              WithdrawalStatus = "failed"
	WithdrawalStatusCancelled           WithdrawalStatus = "cancelled"
	WithdrawalStatusAwaitingAdminReview WithdrawalStatus = "awaiting_admin_review"
)

type Withdrawal struct {
	ID                    string           `json:"id" db:"id"`
	UserID                string           `json:"user_id" db:"user_id" binding:"required"`
	AdminID               string           `json:"admin_id" db:"admin_id"`
	WithdrawalID          string           `json:"withdrawal_id" db:"withdrawal_id" binding:"required"`
	ChainID               string           `json:"chain_id" db:"chain_id" binding:"required"`
	Network               string           `json:"network" db:"network" binding:"required"`
	CryptoCurrency        string           `json:"crypto_currency" db:"crypto_currency" binding:"required"`
	USDAmountCents        int64            `json:"usd_amount_cents" db:"usd_amount_cents" binding:"required"`
	CryptoAmount          string           `json:"crypto_amount" db:"crypto_amount" binding:"required"`
	ExchangeRate          string           `json:"exchange_rate" db:"exchange_rate" binding:"required"`
	FeeCents              int64            `json:"fee_cents" db:"fee_cents"`
	ToAddress             string           `json:"to_address" db:"to_address" binding:"required"`
	TxHash                string           `json:"tx_hash" db:"tx_hash"`
	Status                WithdrawalStatus `json:"status" db:"status" binding:"required"`
	RequiresAdminReview   bool             `json:"requires_admin_review" db:"requires_admin_review"`
	AdminReviewDeadline   time.Time        `json:"admin_review_deadline" db:"admin_review_deadline"`
	ProcessedBySystem     bool             `json:"processed_by_system" db:"processed_by_system"`
	SourceWalletAddress   string           `json:"source_wallet_address" db:"source_wallet_address" binding:"required"`
	AmountReservedCents   int64            `json:"amount_reserved_cents" db:"amount_reserved_cents" binding:"required"`
	ReservationReleased   bool             `json:"reservation_released" db:"reservation_released"`
	ReservationReleasedAt time.Time        `json:"reservation_released_at" db:"reservation_released_at"`
	Metadata              json.RawMessage  `json:"metadata" db:"metadata"`
	CreatedAt             time.Time        `json:"created_at" db:"created_at" binding:"required"`
	UpdatedAt             time.Time        `json:"updated_at" db:"updated_at" binding:"required"`
}
