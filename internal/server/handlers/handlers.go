package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/application/services"
	"github.com/tuncanbit/tvs/internal/server/websocket"
	"github.com/tuncanbit/tvs/pkg/config"
)

type Handlers struct {
	VerificationSvc services.IVerificationService
	Logger          zerolog.Logger
	Config          *config.Config
}

func New(verificationSvc services.IVerificationService, logger zerolog.Logger, config *config.Config) *Handlers {
	return &Handlers{
		VerificationSvc: verificationSvc,
		Logger:          logger,
		Config:          config,
	}
}

func (h *Handlers) SetupHandlers(router *gin.Engine) {
	WebSocketMgr := websocket.NewManager(h.Config.WebSocket)
	verificationHandler := NewVerificationHandler(h.VerificationSvc, WebSocketMgr)
	wsHandler := NewWebSocketHandler(WebSocketMgr)
	healthHandler := NewHealthHandler()

	router.GET("/health", healthHandler.Health)
	router.GET("/ready", healthHandler.Ready)

	// WebSocket endpoint
	router.GET("/status", wsHandler.HandleConnection)

	v1 := router.Group("/v1")
	{
		// Transaction verification routes
		chains := v1.Group("/chain/:chain_id")
		{
			chains.POST("/verify", verificationHandler.VerifyTransaction)
			chains.POST("/verify/batch", verificationHandler.VerifyBatch)
			chains.GET("/transaction/:tx_hash", verificationHandler.GetTransaction)
			chains.GET("/transaction/:tx_hash/status", verificationHandler.GetTransactionStatus)
			chains.GET("/address/:address/transactions", verificationHandler.GetTransactionsByAddress)
		}

		// Transaction management routes
		transactions := v1.Group("/transactions")
		{
			transactions.GET("/:id", verificationHandler.GetTransactionByID)
			transactions.PUT("/:id/reprocess", verificationHandler.ReprocessTransaction)
			transactions.GET("/", verificationHandler.ListTransactions)
		}

		// Statistics and monitoring
		stats := v1.Group("/stats")
		{
			stats.GET("/chain/:chain_id", verificationHandler.GetChainStats)
		}
	}

}
