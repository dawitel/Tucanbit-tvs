package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	gws "github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/server/websocket"
)

var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, implement proper origin checking
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	wsManager interfaces.WebSocketManager
}

func NewWebSocketHandler(wsManager interfaces.WebSocketManager) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
	}
}

func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade WebSocket connection")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Failed to upgrade to WebSocket",
		})
		return
	}

	client := NewWebSocketClient(conn)

	if err := h.wsManager.AddClient(client); err != nil {
		log.Error().Err(err).Str("client_id", client.GetID()).Msg("Failed to add WebSocket client")
		conn.Close()
		return
	}

	log.Info().Str("client_id", client.GetID()).Msg("WebSocket client connected")

	defer func() {
		h.wsManager.RemoveClient(client.GetID())
		client.Close()
		log.Info().Str("client_id", client.GetID()).Msg("WebSocket client disconnected")
	}()

	client.HandleConnection()
}

func NewWebSocketClient(conn *gws.Conn) interfaces.WebSocketClient {
	return websocket.NewClient(conn)
}
