package models

import (
	"encoding/json"
	"time"
)

// ChainType represents supported blockchain types
type ChainType string

const (
	ChainTypeEthereum ChainType = "ethereum"
	ChainTypeSolana   ChainType = "solana"
	ChainTypeBitcoin  ChainType = "bitcoin"
)

// VerificationStatus represents the status of transaction verification
type VerificationStatus string

const (
	StatusPending    VerificationStatus = "pending"
	StatusVerified   VerificationStatus = "verified"
	StatusFailed     VerificationStatus = "failed"
	StatusProcessing VerificationStatus = "processing"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	ID            string             `json:"id" db:"id"`
	ChainID       string             `json:"chain_id" db:"chain_id"`
	ChainType     ChainType          `json:"chain_type" db:"chain_type"`
	TxHash        string             `json:"tx_hash" db:"tx_hash"`
	FromAddress   string             `json:"from_address" db:"from_address"`
	ToAddress     string             `json:"to_address" db:"to_address"`
	Amount        string             `json:"amount" db:"amount"`
	Fee           string             `json:"fee" db:"fee"`
	BlockNumber   *int64             `json:"block_number" db:"block_number"`
	BlockHash     *string            `json:"block_hash" db:"block_hash"`
	Status        VerificationStatus `json:"status" db:"status"`
	Confirmations int                `json:"confirmations" db:"confirmations"`
	Timestamp     time.Time          `json:"timestamp" db:"timestamp"`
	VerifiedAt    *time.Time         `json:"verified_at" db:"verified_at"`
	ProcessorID   *string            `json:"processor_id" db:"processor_id"`
	Metadata      json.RawMessage    `json:"metadata" db:"metadata"`
	CreatedAt     time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" db:"updated_at"`
}

// VerificationRequest represents a request to verify a transaction
type VerificationRequest struct {
	ChainID     string  `json:"chain_id" binding:"required"`
	TxHash      string  `json:"tx_hash" binding:"required"`
	Address     *string `json:"address,omitempty"`
	ProcessorID *string `json:"processor_id,omitempty"`
	Priority    int     `json:"priority,omitempty"`
}

// VerificationResponse represents the response from a verification request
type VerificationResponse struct {
	RequestID      string             `json:"request_id"`
	Transaction    *Transaction       `json:"transaction,omitempty"`
	Status         VerificationStatus `json:"status"`
	Message        string             `json:"message,omitempty"`
	ProcessedAt    time.Time          `json:"processed_at"`
	ProcessingTime time.Duration      `json:"processing_time"`
}

// StatusUpdate represents a real-time status update
type StatusUpdate struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	TxHash    string      `json:"tx_hash,omitempty"`
	ChainID   string      `json:"chain_id,omitempty"`
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// ChainInfo represents blockchain information
type ChainInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      ChainType `json:"type"`
	NetworkID string    `json:"network_id"`
	Active    bool      `json:"active"`
	Endpoints []string  `json:"endpoints"`
}
