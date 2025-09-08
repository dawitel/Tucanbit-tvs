package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/pkg/config"
)

type HeliusClient struct {
	apiKey        string
	baseURLs      map[string]string
	mintAddresses map[string]map[string]string
	httpClient    *http.Client
	logger        zerolog.Logger
}

type VerifyDepositParams struct {
	Address        string
	RequiredAmount int64
	TokenType      domain.SPLTokenType
	ClusterType    domain.SolanaClusterType
}

type VerifyWithdrawalParams struct {
	TxHash      string
	ToAddress   string
	Amount      float64 // Changed to float64 to match API
	TokenType   domain.SPLTokenType
	ClusterType domain.SolanaClusterType
}

func NewHeliusClient(cfg *config.Config, logger zerolog.Logger) *HeliusClient {
	return &HeliusClient{
		apiKey:        cfg.Helius.APIKey,
		baseURLs:      cfg.Helius.BaseURLs,
		mintAddresses: cfg.MintAddresses,
		httpClient: &http.Client{
			Timeout: cfg.Helius.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

func (c *HeliusClient) getBaseURL(clusterType domain.SolanaClusterType) (string, error) {
	url, exists := c.baseURLs[string(clusterType)]
	if !exists {
		return "", fmt.Errorf("no Helius base URL configured for cluster type: %s", clusterType)
	}
	return url, nil
}

func (c *HeliusClient) GetMintAddress(clusterType domain.SolanaClusterType, tokenType domain.SPLTokenType) (string, error) {
	clusterMints, exists := c.mintAddresses[string(clusterType)]
	if !exists {
		return "", fmt.Errorf("no mint addresses configured for cluster type: %s", clusterType)
	}

	mintAddress, exists := clusterMints[string(tokenType)]
	if !exists {
		return "", fmt.Errorf("no mint address configured for token type %s on cluster %s", tokenType, clusterType)
	}

	c.logger.Info().
		Str("cluster", string(clusterType)).
		Str("token", string(tokenType)).
		Str("mint_address", mintAddress).
		Msg("Retrieved mint address")
	return mintAddress, nil
}

func (c *HeliusClient) GetDecimals(clusterType domain.SolanaClusterType, tokenType domain.SPLTokenType) (int, error) {
	if tokenType == domain.SPLTokenTypeSOL {
		return 9, nil
	}
	switch clusterType {
	case domain.SolanaClusterTypeMainnet:
		return 6, nil
	case domain.SolanaClusterTypeTestnet:
		return 9, nil
	default:
		return 0, fmt.Errorf("unsupported cluster type: %s", clusterType)
	}
}

func (c *HeliusClient) GetSignaturesForAddress(ctx context.Context, address string, clusterType domain.SolanaClusterType) ([]domain.HeliusTransaction, error) {
	baseURL, err := c.getBaseURL(clusterType)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v0/addresses/%s/transactions?api-key=%s&type=TRANSFER&limit=100", baseURL, address, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Str("url", url).
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Msg("Helius API request failed")
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	c.logger.Debug().
		Str("url", url).
		Str("response_body", string(body)).
		Msg("Helius API response")

	var transactions []domain.HeliusTransaction
	if err := json.Unmarshal(body, &transactions); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	c.logger.Info().
		Str("address", address).
		Str("cluster", string(clusterType)).
		Int("transaction_count", len(transactions)).
		Msg("Fetched transactions")
	return transactions, nil
}

func (c *HeliusClient) VerifyDeposit(ctx context.Context, params VerifyDepositParams) (bool, []domain.HeliusTransaction, error) {
	c.logger.Info().
		Str("address", params.Address).
		Int64("required_amount", params.RequiredAmount).
		Str("token_type", string(params.TokenType)).
		Str("cluster", string(params.ClusterType)).
		Msg("Starting deposit verification")

	transactions, err := c.GetSignaturesForAddress(ctx, params.Address, params.ClusterType)
	if err != nil {
		return false, nil, fmt.Errorf("failed to fetch transactions: %v", err)
	}

	decimals, err := c.GetDecimals(params.ClusterType, params.TokenType)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get decimals: %v", err)
	}

	var targetMint string
	if params.TokenType != domain.SPLTokenTypeSOL {
		targetMint, err = c.GetMintAddress(params.ClusterType, params.TokenType)
		if err != nil {
			return false, nil, fmt.Errorf("failed to get mint address: %v", err)
		}
	}

	for _, tx := range transactions {
		if tx.Type == "TRANSFER" {
			c.logger.Debug().
				Str("transaction", tx.Signature).
				Int64("fee", tx.Fee).
				Int64("slot", tx.Slot).
				Int64("timestamp", tx.Timestamp).
				Msg("Processing transaction")

			if params.TokenType == domain.SPLTokenTypeSOL {
				for _, transfer := range tx.NativeTransfers {
					c.logger.Debug().
						Str("transaction", tx.Signature).
						Str("from", transfer.FromUserAccount).
						Str("to", transfer.ToUserAccount).
						Int64("amount", transfer.Amount).
						Msg("Inspecting native transfer")
					if transfer.ToUserAccount == params.Address && transfer.Amount >= params.RequiredAmount {
						c.logger.Info().
							Str("transaction", tx.Signature).
							Int64("amount", transfer.Amount).
							Str("token", string(params.TokenType)).
							Str("cluster", string(params.ClusterType)).
							Msg("SOL payment found")
						return true, transactions, nil
					}
				}
			} else {
				for _, transfer := range tx.TokenTransfers {
					adjustedAmount := transfer.TokenAmount // Already in decimal form (e.g., 3.0 for 3 USDC)
					requiredAmountFloat := float64(params.RequiredAmount) / math.Pow(10, float64(decimals))

					c.logger.Debug().
						Str("transaction", tx.Signature).
						Str("from", transfer.FromUserAccount).
						Str("to", transfer.ToUserAccount).
						Str("mint", transfer.Mint).
						Float64("amount", transfer.TokenAmount).
						Msg("Inspecting token transfer")

					if transfer.ToUserAccount == params.Address && transfer.Mint == targetMint && adjustedAmount >= requiredAmountFloat {
						c.logger.Info().
							Str("transaction", tx.Signature).
							Float64("amount", adjustedAmount).
							Str("token", string(params.TokenType)).
							Str("mint", targetMint).
							Str("cluster", string(params.ClusterType)).
							Msg("Token payment found")
						return true, transactions, nil
					}
				}
			}
		}
	}

	c.logger.Warn().
		Str("address", params.Address).
		Int64("required_amount", params.RequiredAmount).
		Str("token_type", string(params.TokenType)).
		Str("cluster", string(params.ClusterType)).
		Str("mint", targetMint).
		Int("transaction_count", len(transactions)).
		Msg("No matching transaction found")
	return false, transactions, fmt.Errorf("no payment of at least %d %s found for address %s on cluster %s",
		params.RequiredAmount, params.TokenType, params.Address, params.ClusterType)
}

func (c *HeliusClient) VerifyWithdrawal(ctx context.Context, params VerifyWithdrawalParams) (bool, domain.HeliusTransaction, error) {
	c.logger.Info().
		Str("tx_hash", params.TxHash).
		Str("to_address", params.ToAddress).
		Float64("amount", params.Amount).
		Str("token_type", string(params.TokenType)).
		Str("cluster", string(params.ClusterType)).
		Msg("Starting withdrawal verification")

	baseURL, err := c.getBaseURL(params.ClusterType)
	if err != nil {
		return false, domain.HeliusTransaction{}, fmt.Errorf("failed to get base URL: %v", err)
	}

	url := fmt.Sprintf("%s/v0/transactions?api-key=%s", baseURL, c.apiKey)
	requestBody := map[string][]string{"transactions": {params.TxHash}}
	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return false, domain.HeliusTransaction{}, fmt.Errorf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return false, domain.HeliusTransaction{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, domain.HeliusTransaction{}, fmt.Errorf("failed to fetch transaction: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Str("url", url).
			Int("status_code", resp.StatusCode).
			Str("response_body", string(responseBody)).
			Msg("Helius API request failed")
		return false, domain.HeliusTransaction{}, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, domain.HeliusTransaction{}, fmt.Errorf("failed to read response body: %v", err)
	}

	c.logger.Debug().
		Str("url", url).
		Str("response_body", string(body)).
		Msg("Helius API response")

	var transactions []domain.HeliusTransaction
	if err := json.Unmarshal(body, &transactions); err != nil {
		return false, domain.HeliusTransaction{}, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if len(transactions) == 0 {
		c.logger.Warn().
			Str("tx_hash", params.TxHash).
			Msg("No transaction found")
		return false, domain.HeliusTransaction{}, fmt.Errorf("no transaction found for hash %s", params.TxHash)
	}

	transaction := transactions[0]
	if transaction.Type != "TRANSFER" {
		c.logger.Warn().
			Str("tx_hash", params.TxHash).
			Str("type", transaction.Type).
			Msg("Transaction is not a transfer")
		return false, transaction, fmt.Errorf("transaction %s is not a transfer", params.TxHash)
	}

	var targetMint string
	if params.TokenType != domain.SPLTokenTypeSOL {
		targetMint, err = c.GetMintAddress(params.ClusterType, params.TokenType)
		if err != nil {
			return false, transaction, fmt.Errorf("failed to get mint address: %v", err)
		}
	}

	if params.TokenType == domain.SPLTokenTypeSOL {
		for _, transfer := range transaction.NativeTransfers {
			c.logger.Debug().
				Str("transaction", transaction.Signature).
				Str("from", transfer.FromUserAccount).
				Str("to", transfer.ToUserAccount).
				Int64("amount", transfer.Amount).
				Msg("Inspecting native transfer")
			if transfer.ToUserAccount == params.ToAddress && float64(transfer.Amount)/1_000_000_000 >= params.Amount {
				c.logger.Info().
					Str("transaction", transaction.Signature).
					Int64("amount", transfer.Amount).
					Str("token", string(params.TokenType)).
					Str("cluster", string(params.ClusterType)).
					Msg("SOL withdrawal found")
				return true, transaction, nil
			}
		}
	} else {
		for _, transfer := range transaction.TokenTransfers {
			c.logger.Debug().
				Str("transaction", transaction.Signature).
				Str("from", transfer.FromUserAccount).
				Str("to", transfer.ToUserAccount).
				Str("mint", transfer.Mint).
				Float64("amount", transfer.TokenAmount).
				Msg("Inspecting token transfer")

			if transfer.ToUserAccount == params.ToAddress && transfer.Mint == targetMint && transfer.TokenAmount >= params.Amount {
				c.logger.Info().
					Str("transaction", transaction.Signature).
					Float64("amount", transfer.TokenAmount).
					Str("token", string(params.TokenType)).
					Str("mint", targetMint).
					Str("cluster", string(params.ClusterType)).
					Msg("Token withdrawal found")
				return true, transaction, nil
			}
		}
	}

	c.logger.Warn().
		Str("tx_hash", params.TxHash).
		Str("to_address", params.ToAddress).
		Float64("amount", params.Amount).
		Str("token_type", string(params.TokenType)).
		Str("cluster", string(params.ClusterType)).
		Str("mint", targetMint).
		Msg("No matching withdrawal found")
	return false, transaction, fmt.Errorf("no withdrawal of at least %f %s found for address %s in transaction %s",
		params.Amount, params.TokenType, params.ToAddress, params.TxHash)
}
