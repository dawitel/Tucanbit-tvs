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
	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/domain/models"
	"github.com/tuncanbit/tvs/pkg/config"
)

type indexingEngineClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
	logger     zerolog.Logger
}

func NewIndexingEngineClient(cfg config.IndexingEngineConfig, logger zerolog.Logger) interfaces.IndexingEngineClient {
	return &indexingEngineClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
		logger:     logger,
	}
}

// GetBlock retrieves block data by hash
func (c *indexingEngineClient) GetBlock(ctx context.Context, chainID, blockHash string) (*models.BlockData, error) {
	endpoint := fmt.Sprintf("/v1/chain/%s/blocks/%s", chainID, blockHash)

	var block models.BlockData
	if err := c.makeRequest(ctx, "GET", endpoint, nil, &block); err != nil {
		return nil, fmt.Errorf("failed to get block %s for chain %s: %w", blockHash, chainID, err)
	}

	return &block, nil
}

// GetBlockByNumber retrieves block data by number
func (c *indexingEngineClient) GetBlockByNumber(ctx context.Context, chainID string, blockNumber int64) (*models.BlockData, error) {
	endpoint := fmt.Sprintf("/v1/chain/%s/blocks", chainID)
	params := url.Values{}
	params.Add("block_number", strconv.FormatInt(blockNumber, 10))

	var blocks []models.BlockData
	if err := c.makeRequestWithParams(ctx, "GET", endpoint, params, &blocks); err != nil {
		return nil, fmt.Errorf("failed to get block %d for chain %s: %w", blockNumber, chainID, err)
	}

	if len(blocks) == 0 {
		return nil, fmt.Errorf("block %d not found for chain %s", blockNumber, chainID)
	}

	return &blocks[0], nil
}

// GetTransaction retrieves transaction data by hash
func (c *indexingEngineClient) GetTransaction(ctx context.Context, chainID, txHash string) (*models.TransactionData, error) {
	endpoint := fmt.Sprintf("/v1/chain/%s/blocks", chainID)
	params := url.Values{}
	params.Add("tx_hash", txHash)

	var blocks []models.BlockData
	if err := c.makeRequestWithParams(ctx, "GET", endpoint, params, &blocks); err != nil {
		return nil, fmt.Errorf("failed to get transaction %s for chain %s: %w", txHash, chainID, err)
	}

	// Search for the transaction in all returned blocks
	for _, block := range blocks {
		for _, tx := range block.Transactions {
			if tx.Hash == txHash {
				return &tx, nil
			}
		}
	}

	return nil, fmt.Errorf("transaction %s not found for chain %s", txHash, chainID)
}

// GetTransactionsByAddress retrieves transactions for an address
func (c *indexingEngineClient) GetTransactionsByAddress(ctx context.Context, chainID, address string, limit, offset int) (*models.AddressTransactions, error) {
	endpoint := fmt.Sprintf("/v1/chain/%s/address/%s/transactions", chainID, address)
	params := url.Values{}
	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Add("offset", strconv.Itoa(offset))
	}

	var addressTxs models.AddressTransactions
	if err := c.makeRequestWithParams(ctx, "GET", endpoint, params, &addressTxs); err != nil {
		return nil, fmt.Errorf("failed to get transactions for address %s on chain %s: %w", address, chainID, err)
	}

	return &addressTxs, nil
}

// GetChainStats retrieves chain statistics
func (c *indexingEngineClient) GetChainStats(ctx context.Context, chainID string) (*models.ChainStats, error) {
	endpoint := fmt.Sprintf("/v1/chain/%s/stats", chainID)

	var stats models.ChainStats
	if err := c.makeRequest(ctx, "GET", endpoint, nil, &stats); err != nil {
		return nil, fmt.Errorf("failed to get stats for chain %s: %w", chainID, err)
	}

	return &stats, nil
}

// makeRequest makes an HTTP request with retries
func (c *indexingEngineClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}, response interface{}) error {
	return c.makeRequestWithParams(ctx, method, endpoint, nil, response)
}

// makeRequestWithParams makes an HTTP request with URL parameters and retries
func (c *indexingEngineClient) makeRequestWithParams(ctx context.Context, method, endpoint string, params url.Values, response interface{}) error {
	fullURL := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(1<<(attempt-1))): // Exponential backoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			log.Warn().Err(err).Int("attempt", attempt+1).Str("url", fullURL).Msg("IE request failed, retrying")
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				continue
			}

			if err := json.Unmarshal(body, response); err != nil {
				lastErr = fmt.Errorf("failed to unmarshal response: %w", err)
				continue
			}

			return nil
		}

		// Handle different HTTP status codes
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
			log.Warn().Int("status", resp.StatusCode).Int("attempt", attempt+1).Str("url", fullURL).Msg("IE server error, retrying")
			continue
		}

		// Client errors (4xx) - don't retry
		return fmt.Errorf("client error (status %d): %s", resp.StatusCode, string(body))
	}

	log.Error().Err(lastErr).Str("url", fullURL).Int("max_retries", c.maxRetries).Msg("IE request failed after all retries")
	return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}
