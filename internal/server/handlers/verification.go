package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/application/services"
	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/domain/models"
)

type VerificationHandler struct {
	verificationService services.IVerificationService
	wsManager           interfaces.WebSocketManager
}

func NewVerificationHandler(verificationService services.IVerificationService, wsManager interfaces.WebSocketManager) *VerificationHandler {
	return &VerificationHandler{
		verificationService: verificationService,
		wsManager:           wsManager,
	}
}

func (h *VerificationHandler) VerifyTransaction(c *gin.Context) {
	chainID := c.Param("chain_id")
	if chainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Chain ID is required",
		})
		return
	}

	var req models.VerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Set chain ID from URL parameter
	req.ChainID = chainID

	// Broadcast status update to WebSocket clients
	h.broadcastStatusUpdate("verification_started", req.TxHash, chainID, "processing", "Verification started")

	response, err := h.verificationService.VerifyTransaction(c.Request.Context(), &req)
	if err != nil {
		log.Error().Err(err).Str("tx_hash", req.TxHash).Msg("Verification service error")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to verify transaction",
		})
		return
	}

	// Broadcast completion status
	h.broadcastStatusUpdate("verification_completed", req.TxHash, chainID, string(response.Status), response.Message)

	c.JSON(http.StatusOK, response)
}

// VerifyBatch handles batch transaction verification
func (h *VerificationHandler) VerifyBatch(c *gin.Context) {
	chainID := c.Param("chain_id")
	if chainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Chain ID is required",
		})
		return
	}

	var requests []*models.VerificationRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": err.Error(),
		})
		return
	}

	// Set chain ID for all requests
	for _, req := range requests {
		req.ChainID = chainID
	}

	// Broadcast batch processing start
	h.broadcastStatusUpdate("batch_verification_started", "", chainID, "processing",
		"Batch verification started for "+strconv.Itoa(len(requests))+" transactions")

	responses, err := h.verificationService.VerifyBatch(c.Request.Context(), requests)
	if err != nil {
		log.Error().Err(err).Str("chain_id", chainID).Msg("Batch verification service error")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to verify batch transactions",
		})
		return
	}

	// Broadcast batch completion
	h.broadcastStatusUpdate("batch_verification_completed", "", chainID, "completed",
		"Batch verification completed for "+strconv.Itoa(len(responses))+" transactions")

	c.JSON(http.StatusOK, gin.H{
		"responses": responses,
		"total":     len(responses),
	})
}

// GetTransaction retrieves a specific transaction
func (h *VerificationHandler) GetTransaction(c *gin.Context) {
	chainID := c.Param("chain_id")
	txHash := c.Param("tx_hash")

	if chainID == "" || txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Chain ID and transaction hash are required",
		})
		return
	}

	transaction, err := h.verificationService.GetTransactionStatus(c.Request.Context(), chainID, txHash)
	if err != nil {
		if err.Error() == "transaction not found: "+txHash {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Transaction not found",
			})
			return
		}

		log.Error().Err(err).Str("chain_id", chainID).Str("tx_hash", txHash).Msg("Failed to get transaction")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve transaction",
		})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// GetTransactionStatus retrieves transaction verification status
func (h *VerificationHandler) GetTransactionStatus(c *gin.Context) {
	chainID := c.Param("chain_id")
	txHash := c.Param("tx_hash")

	if chainID == "" || txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Chain ID and transaction hash are required",
		})
		return
	}

	transaction, err := h.verificationService.GetTransactionStatus(c.Request.Context(), chainID, txHash)
	if err != nil {
		if err.Error() == "transaction not found: "+txHash {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Transaction not found",
			})
			return
		}

		log.Error().Err(err).Str("chain_id", chainID).Str("tx_hash", txHash).Msg("Failed to get transaction status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve transaction status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"chain_id":      transaction.ChainID,
		"tx_hash":       transaction.TxHash,
		"status":        transaction.Status,
		"confirmations": transaction.Confirmations,
		"verified_at":   transaction.VerifiedAt,
	})
}

// GetTransactionsByAddress retrieves transactions for a specific address
func (h *VerificationHandler) GetTransactionsByAddress(c *gin.Context) {
	chainID := c.Param("chain_id")
	address := c.Param("address")

	if chainID == "" || address == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Chain ID and address are required",
		})
		return
	}

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	transactions, err := h.verificationService.GetTransactionsByAddress(c.Request.Context(), chainID, address, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("chain_id", chainID).Str("address", address).Msg("Failed to get transactions by address")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve transactions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
		"total":        len(transactions),
		"limit":        limit,
		"offset":       offset,
	})
}

// GetTransactionByID retrieves a transaction by ID
func (h *VerificationHandler) GetTransactionByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Transaction ID is required",
		})
		return
	}

	transaction, err := h.verificationService.GetTransactionStatus(c.Request.Context(), "", id)
	if err != nil {
		if err.Error() == "transaction not found: "+id {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Transaction not found",
			})
			return
		}

		log.Error().Err(err).Str("id", id).Msg("Failed to get transaction by ID")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to retrieve transaction",
		})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// ReprocessTransaction reprocesses a failed transaction
func (h *VerificationHandler) ReprocessTransaction(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Transaction ID is required",
		})
		return
	}

	response, err := h.verificationService.ReprocessTransaction(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "transaction not found: "+id {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Not Found",
				"message": "Transaction not found",
			})
			return
		}

		log.Error().Err(err).Str("id", id).Msg("Failed to reprocess transaction")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to reprocess transaction",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListTransactions lists transactions with optional filtering
func (h *VerificationHandler) ListTransactions(c *gin.Context) {
	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// For now, return empty list with proper pagination structure
	// This would typically be implemented with a repository method
	c.JSON(http.StatusOK, gin.H{
		"transactions": []interface{}{},
		"total":        0,
		"limit":        limit,
		"offset":       offset,
	})
}

// GetChainStats retrieves statistics for a specific chain
func (h *VerificationHandler) GetChainStats(c *gin.Context) {
	chainID := c.Param("chain_id")
	if chainID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Chain ID is required",
		})
		return
	}

	// For now, return basic stats structure
	// This would typically be implemented with a repository method
	c.JSON(http.StatusOK, gin.H{
		"chain_id":           chainID,
		"total_transactions": 0,
		"verified_count":     0,
		"pending_count":      0,
		"failed_count":       0,
		"processing_count":   0,
		"last_updated":       time.Now(),
	})
}

// broadcastStatusUpdate broadcasts a status update to WebSocket clients
func (h *VerificationHandler) broadcastStatusUpdate(eventType, txHash, chainID, status, message string) {
	update := &models.StatusUpdate{
		Type:      eventType,
		TxHash:    txHash,
		ChainID:   chainID,
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
	}

	if err := h.wsManager.Broadcast(update); err != nil {
		log.Error().Err(err).Msg("Failed to broadcast status update")
	}
}
