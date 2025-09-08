package currency

import (
	"fmt"
	"math"
)

type CurrencyUtils struct{}

func NewCurrencyUtils() *CurrencyUtils {
	return &CurrencyUtils{}
}

// BankersRound applies banker's rounding to a float64 value
func (u *CurrencyUtils) BankersRound(value float64) int64 {
	cents := value * 100
	rounded := math.Round(cents)

	// Check  if we're exactly halfway between two integers
	if math.Abs(cents-rounded) == 0.5 {
		// Banker's rounding: round to nearest even number
		if int64(rounded)%2 == 0 {
			return int64(rounded)
		}
		return int64(rounded) - 1
	}

	return int64(math.Round(cents))
}

// CryptoToUSDCents converts cryptocurrency amount to USD cents using banker's rounding
func (u *CurrencyUtils) CryptoToUSDCents(cryptoAmount float64, exchangeRate float64) int64 {
	usdValue := cryptoAmount * exchangeRate
	return u.BankersRound(usdValue)
}

// CentsToDollars converts cents to dollars for display
func (u *CurrencyUtils) CentsToDollars(cents int64) float64 {
	return float64(cents) / 100.0
}

// FormatUSD formats cents as USD string
func (u *CurrencyUtils) FormatUSD(cents int64) string {
	return fmt.Sprintf("$%.2f", u.CentsToDollars(cents))
}
