package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/domain/models"
	transactionrepository "github.com/tuncanbit/tvs/internal/repositories/transaction_repo"
	"github.com/tuncanbit/tvs/pkg/config"
)

type verificationService struct {
	transactionRepo transactionrepository.ITransactionRepository
	ieClient        interfaces.IndexingEngineClient
	pdmClient       interfaces.PDMClient
	config          config.VerificationConfig
	workerPool      chan struct{}
	logger          zerolog.Logger
}

func NewVerificationService(
	transactionRepo transactionrepository.ITransactionRepository,
	ieClient interfaces.IndexingEngineClient,
	pdmClient interfaces.PDMClient,
	cfg config.VerificationConfig,
	logger zerolog.Logger,
) IVerificationService {
	return &verificationService{
		transactionRepo: transactionRepo,
		ieClient:        ieClient,
		pdmClient:       pdmClient,
		config:          cfg,
		workerPool:      make(chan struct{}, cfg.ConcurrentWorkers),
		logger:          logger,
	}
}

func (s *verificationService) VerifyTransaction(ctx context.Context, req *models.VerificationRequest) (*models.VerificationResponse, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	log.Info().
		Str("request_id", requestID).
		Str("chain_id", req.ChainID).
		Str("tx_hash", req.TxHash).
		Msg("Starting transaction verification")

	// Create timeout context
	verifyCtx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	response := &models.VerificationResponse{
		RequestID:   requestID,
		Status:      models.StatusProcessing,
		ProcessedAt: time.Now(),
	}

	// Check if transaction already exists in cache
	if s.config.CacheEnabled {
		existingTx, err := s.transactionRepo.GetByHash(verifyCtx, req.ChainID, req.TxHash)
		if err != nil {
			log.Error().Err(err).Str("request_id", requestID).Msg("Failed to check existing transaction")
		} else if existingTx != nil && existingTx.Status == models.StatusVerified {
			// Return cached result if not expired
			if time.Since(*existingTx.VerifiedAt) < s.config.CacheTTL {
				response.Transaction = existingTx
				response.Status = models.StatusVerified
				response.ProcessingTime = time.Since(startTime)
				return response, nil
			}
		}
	}

	// Verify transaction
	transaction, err := s.performVerification(verifyCtx, req)
	if err != nil {
		response.Status = models.StatusFailed
		response.Message = err.Error()
		response.ProcessingTime = time.Since(startTime)

		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("tx_hash", req.TxHash).
			Msg("Transaction verification failed")

		return response, nil
	}

	response.Transaction = transaction
	response.Status = models.StatusVerified
	response.ProcessingTime = time.Since(startTime)

	log.Info().
		Str("request_id", requestID).
		Str("tx_hash", req.TxHash).
		Dur("processing_time", response.ProcessingTime).
		Msg("Transaction verification completed")

	return response, nil
}

// VerifyBatch verifies multiple transactions concurrently
func (s *verificationService) VerifyBatch(ctx context.Context, requests []*models.VerificationRequest) ([]*models.VerificationResponse, error) {
	if len(requests) == 0 {
		return []*models.VerificationResponse{}, nil
	}

	responses := make([]*models.VerificationResponse, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, req := range requests {
		wg.Add(1)
		go func(index int, request *models.VerificationRequest) {
			defer wg.Done()

			// Acquire worker slot
			s.workerPool <- struct{}{}
			defer func() { <-s.workerPool }()

			response, err := s.VerifyTransaction(ctx, request)
			if err != nil {
				response = &models.VerificationResponse{
					RequestID:      uuid.New().String(),
					Status:         models.StatusFailed,
					Message:        err.Error(),
					ProcessedAt:    time.Now(),
					ProcessingTime: 0,
				}
			}

			mu.Lock()
			responses[index] = response
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()
	return responses, nil
}

// GetTransactionStatus gets the current status of a transaction
func (s *verificationService) GetTransactionStatus(ctx context.Context, chainID, txHash string) (*models.Transaction, error) {
	transaction, err := s.transactionRepo.GetByHash(ctx, chainID, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction status: %w", err)
	}

	if transaction == nil {
		return nil, fmt.Errorf("transaction not found: %s", txHash)
	}

	return transaction, nil
}

// GetTransactionsByAddress retrieves transactions for an address
func (s *verificationService) GetTransactionsByAddress(ctx context.Context, chainID, address string, limit, offset int) ([]*models.Transaction, error) {
	transactions, err := s.transactionRepo.GetByAddress(ctx, chainID, address, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by address: %w", err)
	}

	return transactions, nil
}

// ReprocessTransaction reprocesses a failed transaction
func (s *verificationService) ReprocessTransaction(ctx context.Context, id string) (*models.VerificationResponse, error) {
	transaction, err := s.transactionRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction for reprocessing: %w", err)
	}

	if transaction == nil {
		return nil, fmt.Errorf("transaction not found: %s", id)
	}

	// Create verification request from stored transaction
	req := &models.VerificationRequest{
		ChainID:     transaction.ChainID,
		TxHash:      transaction.TxHash,
		ProcessorID: transaction.ProcessorID,
	}

	return s.VerifyTransaction(ctx, req)
}

// performVerification performs the actual transaction verification
func (s *verificationService) performVerification(ctx context.Context, req *models.VerificationRequest) (*models.Transaction, error) {
	// Step 1: Retrieve transaction data from Indexing Engine
	ieTransaction, err := s.ieClient.GetTransaction(ctx, req.ChainID, req.TxHash)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve transaction from IE: %w", err)
	}

	// Step 2: Convert IE transaction to domain transaction
	transaction := s.convertIETransactionToDomain(req.ChainID, ieTransaction)

	// Step 3: Perform processor-specific verification if required
	if req.ProcessorID != nil && *req.ProcessorID != "" {
		if err := s.performProcessorVerification(ctx, transaction, *req.ProcessorID); err != nil {
			transaction.Status = models.StatusFailed
			// Continue to store the failed verification
		} else {
			transaction.Status = models.StatusVerified
			now := time.Now()
			transaction.VerifiedAt = &now
		}
		transaction.ProcessorID = req.ProcessorID
	} else {
		// Basic verification - check if transaction exists and has confirmations
		if ieTransaction.Confirmations > 0 {
			transaction.Status = models.StatusVerified
			now := time.Now()
			transaction.VerifiedAt = &now
		} else {
			transaction.Status = models.StatusPending
		}
	}

	// Step 4: Store or update transaction in database
	existingTx, err := s.transactionRepo.GetByHash(ctx, req.ChainID, req.TxHash)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check existing transaction")
	}

	if existingTx != nil {
		// Update existing transaction
		transaction.ID = existingTx.ID
		transaction.CreatedAt = existingTx.CreatedAt

		if err := s.transactionRepo.Update(ctx, transaction); err != nil {
			return nil, fmt.Errorf("failed to update transaction: %w", err)
		}
	} else {
		// Create new transaction
		if err := s.transactionRepo.Create(ctx, transaction); err != nil {
			return nil, fmt.Errorf("failed to store transaction: %w", err)
		}
	}

	return transaction, nil
}

// performProcessorVerification performs processor-specific verification via PDM
func (s *verificationService) performProcessorVerification(ctx context.Context, tx *models.Transaction, processorID string) error {
	// Create processor request payload based on chain type and processor
	payload, err := s.createProcessorPayload(tx, processorID)
	if err != nil {
		return fmt.Errorf("failed to create processor payload: %w", err)
	}

	// Send request to PDM
	response, err := s.pdmClient.SendRequest(ctx, processorID, payload)
	if err != nil {
		return fmt.Errorf("PDM request failed: %w", err)
	}

	// Process response based on processor type
	if err := s.processProcessorResponse(tx, processorID, response); err != nil {
		return fmt.Errorf("failed to process processor response: %w", err)
	}

	return nil
}

// createProcessorPayload creates a processor-specific request payload
func (s *verificationService) createProcessorPayload(tx *models.Transaction, processorID string) (*models.ProcessorRequestPayload, error) {
	payload := &models.ProcessorRequestPayload{
		RequestType: "verification",
		Method:      "POST",
		Timeout:     30,
	}

	// Create processor-specific payload based on chain type and processor ID
	switch tx.ChainType {
	case models.ChainTypeBitcoin:
		if processorID == "btcpay" {
			btcRequest := &models.BTCPayServerRequest{
				TransactionID: tx.TxHash,
				Amount:        tx.Amount,
				Confirmations: tx.Confirmations,
			}

			body, err := json.Marshal(btcRequest)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal BTC Pay request: %w", err)
			}

			payload.Body = body
			payload.Endpoint = "/api/v1/transactions/verify"
		}

	case models.ChainTypeEthereum, models.ChainTypeSolana:
		// Generic verification payload for other chains
		genericRequest := map[string]interface{}{
			"transaction_hash": tx.TxHash,
			"chain_id":         tx.ChainID,
			"amount":           tx.Amount,
			"from_address":     tx.FromAddress,
			"to_address":       tx.ToAddress,
			"confirmations":    tx.Confirmations,
		}

		body, err := json.Marshal(genericRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal generic verification request: %w", err)
		}

		payload.Body = body
		payload.Endpoint = "/api/v1/verify"
	}

	return payload, nil
}

// processProcessorResponse processes the response from a processor
func (s *verificationService) processProcessorResponse(tx *models.Transaction, processorID string, response *models.ProcessorResponse) error {
	if response.Status != "completed" {
		return fmt.Errorf("processor verification failed: %s", response.Error)
	}

	// Process response based on processor type
	switch processorID {
	case "btcpay":
		var btcResponse models.BTCPayServerResponse
		if err := json.Unmarshal(response.Data, &btcResponse); err != nil {
			return fmt.Errorf("failed to unmarshal BTC Pay response: %w", err)
		}

		if !btcResponse.Confirmed {
			return fmt.Errorf("transaction not confirmed by BTC Pay Server")
		}

		// Update transaction with processor data
		tx.Confirmations = btcResponse.Confirmations
		if btcResponse.Fee != "" {
			tx.Fee = btcResponse.Fee
		}

	default:
		// Generic processor response handling
		var genericResponse map[string]interface{}
		if err := json.Unmarshal(response.Data, &genericResponse); err != nil {
			return fmt.Errorf("failed to unmarshal processor response: %w", err)
		}

		confirmed, ok := genericResponse["confirmed"].(bool)
		if !ok || !confirmed {
			return fmt.Errorf("transaction not confirmed by processor")
		}
	}

	// Store processor response in metadata
	metadata := map[string]interface{}{
		"processor_id":       processorID,
		"processor_response": response,
		"verified_at":        time.Now(),
	}

	metadataJSON, _ := json.Marshal(metadata)
	tx.Metadata = metadataJSON

	return nil
}

// convertIETransactionToDomain converts IE transaction data to domain transaction
func (s *verificationService) convertIETransactionToDomain(chainID string, ieTransaction *models.TransactionData) *models.Transaction {
	// Determine chain type from chain ID
	chainType := s.determineChainType(chainID)

	transaction := &models.Transaction{
		ID:            uuid.New().String(),
		ChainID:       chainID,
		ChainType:     chainType,
		TxHash:        ieTransaction.Hash,
		FromAddress:   ieTransaction.From,
		ToAddress:     ieTransaction.To,
		Amount:        ieTransaction.Value,
		Fee:           ieTransaction.Fee,
		BlockNumber:   &ieTransaction.BlockNumber,
		BlockHash:     &ieTransaction.BlockHash,
		Status:        models.StatusPending,
		Confirmations: ieTransaction.Confirmations,
		Timestamp:     ieTransaction.Timestamp,
		Metadata:      ieTransaction.Metadata,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return transaction
}

// determineChainType determines the chain type from chain ID
func (s *verificationService) determineChainType(chainID string) models.ChainType {
	switch {
	case chainID == "eth-mainnet" || chainID == "eth-goerli" || chainID == "eth-sepolia":
		return models.ChainTypeEthereum
	case chainID == "sol-mainnet" || chainID == "sol-devnet":
		return models.ChainTypeSolana
	case chainID == "btc-mainnet" || chainID == "btc-testnet":
		return models.ChainTypeBitcoin
	default:
		return models.ChainTypeEthereum // Default fallback
	}
}
