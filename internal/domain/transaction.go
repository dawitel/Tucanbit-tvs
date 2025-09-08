package domain

import (
	"encoding/json"
	"time"
)

type VerificationStatus string
type ProcessorType string
type TransactionType string

const (
	StatusPending    VerificationStatus = "pending"
	StatusVerified   VerificationStatus = "verified"
	StatusFailed     VerificationStatus = "failed"
	StatusProcessing VerificationStatus = "processing"
)

const (
	ProcessorInternal ProcessorType = "internal"
	ProcessorPDM      ProcessorType = "pdm"
)

const (
	TypeDeposit    TransactionType = "deposit"
	TypeWithdrawal TransactionType = "withdrawal"
)

type Transaction struct {
	ID               string             `json:"id" db:"id"`
	DepositSessionID string             `json:"deposit_session_id" db:"deposit_session_id"`
	WithdrawalID     string             `json:"withdrawal_id" db:"withdrawal_id"`
	ChainID          string             `json:"chain_id" db:"chain_id" binding:"required"`
	Network          string             `json:"network" db:"network" binding:"required"`
	CryptoCurrency   string             `json:"crypto_currency" db:"crypto_currency" binding:"required"`
	TxHash           string             `json:"tx_hash" db:"tx_hash" binding:"required"`
	FromAddress      string             `json:"from_address" db:"from_address" binding:"required"`
	ToAddress        string             `json:"to_address" db:"to_address" binding:"required"`
	Amount           string             `json:"amount" db:"amount" binding:"required"`
	USDAmountCents   int64              `json:"usd_amount_cents" db:"usd_amount_cents"`
	ExchangeRate     string             `json:"exchange_rate" db:"exchange_rate"`
	Fee              string             `json:"fee" db:"fee"`
	BlockNumber      int64              `json:"block_number" db:"block_number"`
	BlockHash        string             `json:"block_hash" db:"block_hash"`
	Status           VerificationStatus `json:"status" db:"status" binding:"required"`
	Confirmations    int                `json:"confirmations" db:"confirmations"`
	Timestamp        time.Time          `json:"timestamp" db:"timestamp" binding:"required"`
	VerifiedAt       time.Time          `json:"verified_at" db:"verified_at"`
	Processor        ProcessorType      `json:"processor" db:"processor" binding:"required"`
	TransactionType  TransactionType    `json:"transaction_type" db:"transaction_type" binding:"required"`
	Metadata         json.RawMessage    `json:"metadata" db:"metadata"`
	CreatedAt        time.Time          `json:"created_at" db:"created_at" binding:"required"`
	UpdatedAt        time.Time          `json:"updated_at" db:"updated_at" binding:"required"`
}
