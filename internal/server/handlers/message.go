package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/tuncanbit/tvs/internal/domain"
	"github.com/tuncanbit/tvs/internal/server/websocket"
)

type MessageHandler struct {
	wsHub *websocket.WsHub
}

func NewMessageHandler(wsHub *websocket.WsHub) *MessageHandler {
	return &MessageHandler{
		wsHub: wsHub,
	}
}

type WsMessage struct {
	Type       string                 `json:"type"`
	Deposit    *domain.DepositSession `json:"deposit,omitempty"`
	Withdrawal *domain.Withdrawal     `json:"withdrawal,omitempty"`
	Balance    *domain.Balance        `json:"balance,omitempty"`
}

func (mh *MessageHandler) HandleMessage(c *gin.Context) {
	var msg WsMessage
	if err := c.BindJSON(&msg); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	switch msg.Type {
	case "deposit":
		if msg.Deposit != nil {
			mh.wsHub.BroadcastDepositSession(*msg.Deposit)
		}
	case "withdrawal":
		if msg.Withdrawal != nil {
			mh.wsHub.BroadcastWithdrawal(*msg.Withdrawal)
		}
	case "balance":
		if msg.Balance != nil {
			mh.wsHub.BroadcastBalance(*msg.Balance)
		}
	default:
		c.JSON(400, gin.H{"error": "invalid message type"})
	}

}
