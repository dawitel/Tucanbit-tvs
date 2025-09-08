package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/server/websocket"
)

type SessionStatusHandler struct {
	logger zerolog.Logger
	wsHub  *websocket.WsHub
}

func NewSessionStatusHandler(wsHub *websocket.WsHub, logger zerolog.Logger) *SessionStatusHandler {
	return &SessionStatusHandler{
		logger: logger,
		wsHub:  wsHub,
	}
}

var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *SessionStatusHandler) HandleWebSocket(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		h.logger.Error().Msg("User ID not found in context")
		c.JSON(http.StatusUnauthorized, domain.ApiResponse{
			Message: "User not authenticated",
			Success: false,
			Status:  http.StatusUnauthorized,
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		h.logger.Error().Err(err).Msg("Invalid user ID format")
		c.JSON(http.StatusBadRequest, domain.ApiResponse{
			Message: "Invalid user ID format",
			Success: false,
			Status:  http.StatusBadRequest,
		})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Err(err).
			Str("user_id", userID.String()).
			Msg("Failed to upgrade to WebSocket")
		c.JSON(http.StatusInternalServerError, domain.ApiResponse{
			Message: "Failed to establish WebSocket connection: " + err.Error(),
			Success: false,
			Status:  http.StatusInternalServerError,
		})
		return
	}

	client := &websocket.WsClient{
		UserID: userID.String(),
		Conn:   conn,
	}
	h.wsHub.Register <- client
	h.logger.Info().
		Str("user_id", userID.String()).
		Msg("WebSocket client registration sent")

	go func() {
		defer func() {
			h.wsHub.Unregister <- client
		}()

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				h.logger.Err(err).
					Str("user_id", userID.String()).
					Msg("WebSocket read error")
				break
			}
		}
	}()

}

func (h *SessionStatusHandler) TestWebSocket(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		h.logger.Error().Msg("User ID not found in context")
		c.JSON(http.StatusUnauthorized, domain.ApiResponse{
			Message: "User not authenticated",
			Success: false,
			Status:  http.StatusUnauthorized,
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		h.logger.Error().Err(err).Msg("Invalid user ID format")
		c.JSON(http.StatusBadRequest, domain.ApiResponse{
			Message: "Invalid user ID format",
			Success: false,
			Status:  http.StatusBadRequest,
		})
		return
	}

	session := domain.DepositSession{
		ID:             uuid.New().String(),
		ErrorMessage:   "",
		QRCodeData:     "",
		PaymentLink:    "s string",
		SessionID:      "DEP-W227B0D7W9",
		UserID:         userID.String(),
		ChainID:        "sol-testnet",
		Network:        "SOL",
		CryptoCurrency: "SOL",
		Amount:         1,
		Status:         domain.SessionStatusFailed,
		WalletAddress:  "test-address",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	h.logger.Info().
		Str("session_id", session.SessionID).
		Str("user_id", session.UserID).
		Msg("Test broadcast sent")
	h.wsHub.BroadcastDepositSession(session)

}
