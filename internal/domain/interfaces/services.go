package interfaces

import (
	"context"

	"github.com/tuncanbit/tvs/internal/domain/models"
)

type IndexingEngineClient interface {
	// GetBlock retrieves block data by hash
	GetBlock(ctx context.Context, chainID, blockHash string) (*models.BlockData, error)

	// GetBlockByNumber retrieves block data by number
	GetBlockByNumber(ctx context.Context, chainID string, blockNumber int64) (*models.BlockData, error)

	// GetTransaction retrieves transaction data by hash
	GetTransaction(ctx context.Context, chainID, txHash string) (*models.TransactionData, error)

	// GetTransactionsByAddress retrieves transactions for an address
	GetTransactionsByAddress(ctx context.Context, chainID, address string, limit, offset int) (*models.AddressTransactions, error)

	// GetChainStats retrieves chain statistics
	GetChainStats(ctx context.Context, chainID string) (*models.ChainStats, error)
}

// PDMClient defines the interface for PDM communication
type PDMClient interface {
	// SendRequest sends a request to a processor via PDM
	SendRequest(ctx context.Context, processorID string, payload *models.ProcessorRequestPayload) (*models.ProcessorResponse, error)

	// GetProcessorInfo retrieves processor information
	GetProcessorInfo(ctx context.Context, processorID string) (*models.ProcessorInfo, error)

	// ListProcessors lists available processors
	ListProcessors(ctx context.Context) ([]*models.ProcessorInfo, error)
}

// WebSocketManager defines the interface for WebSocket management
type WebSocketManager interface {
	AddClient(client WebSocketClient) error
	RemoveClient(clientID string) error
	Broadcast(message *models.StatusUpdate) error
	SendToClient(clientID string, message *models.StatusUpdate) error
	GetClientCount() int
}

type WebSocketClient interface {
	GetID() string
	Send(message *models.StatusUpdate) error
	Close() error
	IsActive() bool
	HandleConnection()
}
