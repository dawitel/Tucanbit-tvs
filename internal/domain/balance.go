package domain

import (
	"time"
)

type Balance struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id" binding:"required"`
	CurrencyCode  string    `json:"currency_code" db:"currency_code" binding:"required"`
	AmountCents   int64     `json:"amount_cents" db:"amount_cents"`
	AmountUnits   string    `json:"amount_units" db:"amount_units"`
	ReservedCents int64     `json:"reserved_cents" db:"reserved_cents"`
	ReservedUnits string    `json:"reserved_units" db:"reserved_units"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at" binding:"required"`
}

type BalanceLog struct {
	ID                 string
	UserID             string
	Component          string
	CurrencyCode       string
	ChangeCents        int64
	ChangeUnits        float64
	OperationalGroupID string
	OperationalTypeID  string
	Description        string
	Timestamp          time.Time
	BalanceAfterCents  int64
	BalanceAfterUnits  float64
	TransactionID      string
	Status             string
}
