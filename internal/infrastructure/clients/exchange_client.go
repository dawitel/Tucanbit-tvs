// pkg/clients/exchange_client.go
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/pkg/config"
)

type ExchangeAPIClient struct {
	baseURL    string
	httpClient *http.Client
	config     *config.ExchangeAPIConfig
	logger     zerolog.Logger
}

func NewExchangeAPIClient(cfg *config.ExchangeAPIConfig, logger zerolog.Logger) *ExchangeAPIClient {
	return &ExchangeAPIClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 10,
			},
		},
		config: cfg,
		logger: logger.With().Str("component", "coincap_api_client").Logger(),
	}
}

func (c *ExchangeAPIClient) GetExchangeRate(ctx context.Context, cryptoCurrency, fiatCurrency string) (*domain.ExchangeRateResponse, error) {
	return c.getExchangeRateWithRetry(ctx, cryptoCurrency, 0)
}

func (c *ExchangeAPIClient) GetMultipleExchangeRates(ctx context.Context, cryptoCurrencies []string, fiatCurrency string) (map[string]*domain.ExchangeRateResponse, error) {
	rates := make(map[string]*domain.ExchangeRateResponse)

	for _, crypto := range cryptoCurrencies {
		rate, err := c.GetExchangeRate(ctx, crypto, fiatCurrency)
		if err != nil {
			c.logger.Warn().Err(err).Str("crypto", crypto).Msg("Failed to get exchange rate for cryptocurrency")
			continue
		}
		rates[crypto] = rate
	}

	return rates, nil
}

func (c *ExchangeAPIClient) GetExchangeRateWithTimestamp(ctx context.Context, cryptoCurrency, fiatCurrency string, timestamp time.Time) (*domain.ExchangeRateResponse, error) {
	return c.GetExchangeRate(ctx, cryptoCurrency, fiatCurrency)
}

func (c *ExchangeAPIClient) getExchangeRateWithRetry(ctx context.Context, cryptoCurrency string, attempt int) (*domain.ExchangeRateResponse, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	coinCapID := c.mapCryptoToCoinCapID(cryptoCurrency)
	u.Path = fmt.Sprintf("/v3/assets/%s", coinCapID)

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request failed: %w", err)
	}

	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if shouldRetry(err) && attempt < c.config.MaxRetries {
			backoff := calculateBackoff(attempt, c.config.RetryBackoffBase)
			c.logger.Info().
				Err(err).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Request failed, retrying after backoff")

			time.Sleep(backoff)
			return c.getExchangeRateWithRetry(ctx, cryptoCurrency, attempt+1)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if shouldRetryStatusCode(resp.StatusCode) && attempt < c.config.MaxRetries {
			backoff := calculateBackoff(attempt, c.config.RetryBackoffBase)
			c.logger.Warn().
				Int("status_code", resp.StatusCode).
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Received non-200 status, retrying after backoff")

			time.Sleep(backoff)
			return c.getExchangeRateWithRetry(ctx, cryptoCurrency, attempt+1)
		}
		return nil, c.handleErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}

	return c.parseCoinCapResponse(body, cryptoCurrency)
}

func (c *ExchangeAPIClient) parseCoinCapResponse(body []byte, cryptoCurrency string) (*domain.ExchangeRateResponse, error) {
	var response struct {
		Data      domain.CoinCapAsset `json:"data"`
		Timestamp int64               `json:"timestamp"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("parsing JSON response failed: %w", err)
	}

	price, err := strconv.ParseFloat(response.Data.PriceUSD, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid price format: %w", err)
	}

	change24Hr, err := strconv.ParseFloat(response.Data.ChangePercent24Hr, 64)
	if err != nil {
		change24Hr = 0
	}

	lastUpdated := time.Unix(response.Timestamp/1000, 0).Format(time.RFC3339)

	return &domain.ExchangeRateResponse{
		CryptoCurrency: cryptoCurrency,
		FiatCurrency:   "USD",
		Rate:           price,
		PriceUSD:       price,
		Change24Hr:     change24Hr,
		LastUpdated:    lastUpdated,
	}, nil
}

func (c *ExchangeAPIClient) handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP error %d: failed to read response body", resp.StatusCode)
	}

	var errorResp struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, errorResp.Error)
	}

	return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
}

func (c *ExchangeAPIClient) mapCryptoToCoinCapID(cryptoCurrency string) string {
	mapping := map[string]string{
		"BTC":   "bitcoin",
		"ETH":   "ethereum",
		"LTC":   "litecoin",
		"USDT":  "tether",
		"USDC":  "usd-coin",
		"BNB":   "binance-coin",
		"XRP":   "xrp",
		"ADA":   "cardano",
		"DOGE":  "dogecoin",
		"SOL":   "solana",
		"DOT":   "polkadot",
		"MATIC": "polygon",
		"AVAX":  "avalanche",
		"BUSD":  "binance-usd",
		"DAI":   "dai",
		"SHIB":  "shiba-inu",
		"TRX":   "tron",
		"UNI":   "uniswap",
		"LINK":  "chainlink",
		"ATOM":  "cosmos",
	}

	if id, exists := mapping[cryptoCurrency]; exists {
		return id
	}

	return cryptoCurrency
}

func shouldRetry(err error) bool {
	if err, ok := err.(interface{ Timeout() bool }); ok && err.Timeout() {
		return true
	}
	if err, ok := err.(interface{ Temporary() bool }); ok && err.Temporary() {
		return true
	}
	return false
}

func shouldRetryStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

func calculateBackoff(attempt, base int) time.Duration {
	if base <= 0 {
		base = 2
	}
	backoff := time.Duration(base^attempt) * time.Second
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}
	return backoff
}
