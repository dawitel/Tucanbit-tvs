package models

import (
	"encoding/json"
	"time"
)

// IE API Response Models

// BlockData represents block information from the Indexing Engine
type BlockData struct {
	Hash         string            `json:"hash"`
	Number       int64             `json:"number"`
	Timestamp    time.Time         `json:"timestamp"`
	ParentHash   string            `json:"parent_hash"`
	Transactions []TransactionData `json:"transactions"`
	Metadata     json.RawMessage   `json:"metadata,omitempty"`
}

// TransactionData represents transaction data from the Indexing Engine
type TransactionData struct {
	Hash             string          `json:"hash"`
	From             string          `json:"from"`
	To               string          `json:"to"`
	Value            string          `json:"value"`
	Gas              string          `json:"gas,omitempty"`
	GasPrice         string          `json:"gas_price,omitempty"`
	Fee              string          `json:"fee,omitempty"`
	Status           string          `json:"status"`
	BlockNumber      int64           `json:"block_number"`
	BlockHash        string          `json:"block_hash"`
	TransactionIndex int             `json:"transaction_index"`
	Confirmations    int             `json:"confirmations"`
	Timestamp        time.Time       `json:"timestamp"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

// AddressTransactions represents transactions for a specific address
type AddressTransactions struct {
	Address      string            `json:"address"`
	Transactions []TransactionData `json:"transactions"`
	Total        int               `json:"total"`
	Page         int               `json:"page"`
	Limit        int               `json:"limit"`
}

// ChainStats represents chain statistics from the IE
type ChainStats struct {
	ChainID     string    `json:"chain_id"`
	LatestBlock int64     `json:"latest_block"`
	LatestHash  string    `json:"latest_hash"`
	TotalTxs    int64     `json:"total_transactions"`
	LastUpdated time.Time `json:"last_updated"`
}
