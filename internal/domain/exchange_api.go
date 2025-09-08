package domain

type CoinCapResponse struct {
	Data      []CoinCapAsset `json:"data"`
	Timestamp int64          `json:"timestamp"`
}

type CoinCapAsset struct {
	ID                string `json:"id"`
	Rank              string `json:"rank"`
	Symbol            string `json:"symbol"`
	Name              string `json:"name"`
	Supply            string `json:"supply"`
	MaxSupply         string `json:"maxSupply"`
	MarketCapUSD      string `json:"marketCapUsd"`
	VolumeUSD24Hr     string `json:"volumeUsd24Hr"`
	PriceUSD          string `json:"priceUsd"`
	ChangePercent24Hr string `json:"changePercent24Hr"`
	VWAP24Hr          string `json:"vwap24Hr"`
}

type ExchangeRateResponse struct {
	CryptoCurrency string  `json:"crypto_currency"`
	FiatCurrency   string  `json:"fiat_currency"`
	Rate           float64 `json:"rate"`
	LastUpdated    string  `json:"last_updated"`
	PriceUSD       float64 `json:"price_usd"`
	Change24Hr     float64 `json:"change_24hr"`
}
