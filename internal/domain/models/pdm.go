package models

import (
	"encoding/json"
	"time"
)

// PDM API Request/Response Models

// ProcessorRequestPayload represents the payload sent to PDM
type ProcessorRequestPayload struct {
	RequestType string            `json:"request_type"` // verification, confirmation, etc.
	Method      string            `json:"method"`       // GET, POST, PUT, etc.
	Endpoint    string            `json:"endpoint,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        json.RawMessage   `json:"body,omitempty"`
	Timeout     int               `json:"timeout,omitempty"` // seconds
}

// ProcessorResponse represents the response from PDM
type ProcessorResponse struct {
	RequestID      string          `json:"request_id"`
	ProcessorID    string          `json:"processor_id"`
	Status         string          `json:"status"` // completed, failed, timeout
	StatusCode     int             `json:"status_code,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	Error          string          `json:"error,omitempty"`
	ProcessingTime time.Duration   `json:"processing_time"`
	Timestamp      time.Time       `json:"timestamp"`
}

// ProcessorInfo represents processor information
type ProcessorInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Active       bool                   `json:"active"`
	Endpoints    []string               `json:"endpoints"`
	Capabilities []string               `json:"capabilities"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// BTC Pay Server specific models

// BTCPayServerRequest represents a request to BTC Pay Server via PDM
type BTCPayServerRequest struct {
	TransactionID string `json:"transaction_id"`
	Address       string `json:"address,omitempty"`
	Amount        string `json:"amount,omitempty"`
	Confirmations int    `json:"confirmations,omitempty"`
}

// BTCPayServerResponse represents BTC Pay Server response
type BTCPayServerResponse struct {
	TransactionID string    `json:"transaction_id"`
	Confirmed     bool      `json:"confirmed"`
	Confirmations int       `json:"confirmations"`
	Amount        string    `json:"amount"`
	Fee           string    `json:"fee,omitempty"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}
