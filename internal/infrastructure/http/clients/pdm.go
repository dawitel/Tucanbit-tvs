package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/domain/models"
	"github.com/tuncanbit/tvs/pkg/config"
)

type pdmClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
	logger     zerolog.Logger
}

func NewPDMClient(cfg config.PDMConfig, logger zerolog.Logger) interfaces.PDMClient {
	return &pdmClient{
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

func (c *pdmClient) SendRequest(ctx context.Context, processorID string, payload *models.ProcessorRequestPayload) (*models.ProcessorResponse, error) {
	endpoint := fmt.Sprintf("/v1/processor/%s/request", processorID)

	var response models.ProcessorResponse
	if err := c.makeRequest(ctx, "POST", endpoint, payload, &response); err != nil {
		return nil, fmt.Errorf("failed to send request to processor %s: %w", processorID, err)
	}

	return &response, nil
}

func (c *pdmClient) GetProcessorInfo(ctx context.Context, processorID string) (*models.ProcessorInfo, error) {
	endpoint := fmt.Sprintf("/v1/processor/%s", processorID)

	var info models.ProcessorInfo
	if err := c.makeRequest(ctx, "GET", endpoint, nil, &info); err != nil {
		return nil, fmt.Errorf("failed to get processor info for %s: %w", processorID, err)
	}

	return &info, nil
}

// ListProcessors lists available processors
func (c *pdmClient) ListProcessors(ctx context.Context) ([]*models.ProcessorInfo, error) {
	endpoint := "/v1/processors"

	var processors []*models.ProcessorInfo
	if err := c.makeRequest(ctx, "GET", endpoint, nil, &processors); err != nil {
		return nil, fmt.Errorf("failed to list processors: %w", err)
	}

	return processors, nil
}

// makeRequest makes an HTTP request with retries
func (c *pdmClient) makeRequest(ctx context.Context, method, endpoint string, body interface{}, response interface{}) error {
	fullURL := c.baseURL + endpoint

	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(1<<(attempt-1))): // Exponential backoff
			}
		}

		var reqBody []byte
		var err error

		if body != nil {
			reqBody, err = json.Marshal(body)
			if err != nil {
				return fmt.Errorf("failed to marshal request body: %w", err)
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(reqBody))
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
			log.Warn().Err(err).Int("attempt", attempt+1).Str("url", fullURL).Msg("PDM request failed, retrying")
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				continue
			}

			if response != nil {
				if err := json.Unmarshal(respBody, response); err != nil {
					lastErr = fmt.Errorf("failed to unmarshal response: %w", err)
					continue
				}
			}

			return nil
		}

		// Handle different HTTP status codes
		respBody, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBody))
			log.Warn().Int("status", resp.StatusCode).Int("attempt", attempt+1).Str("url", fullURL).Msg("PDM server error, retrying")
			continue
		}

		// Client errors (4xx) - don't retry
		return fmt.Errorf("client error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Error().Err(lastErr).Str("url", fullURL).Int("max_retries", c.maxRetries).Msg("PDM request failed after all retries")
	return fmt.Errorf("request failed after %d retries: %w", c.maxRetries, lastErr)
}
