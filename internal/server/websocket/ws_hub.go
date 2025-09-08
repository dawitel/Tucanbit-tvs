package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/tuncanbit/tvs/internal/domain"
)

type WsHub struct {
	Clients    map[string]map[*websocket.Conn]bool
	Broadcast  chan WsMessage
	Register   chan *WsClient
	Unregister chan *WsClient
	Logger     zerolog.Logger
}

type WsClient struct {
	UserID string
	Conn   *websocket.Conn
}

type WsMessage struct {
	Type       string                 `json:"type"`
	Deposit    *domain.DepositSession `json:"deposit,omitempty"`
	Withdrawal *domain.Withdrawal     `json:"withdrawal,omitempty"`
	Balance    *domain.Balance        `json:"balance,omitempty"`
}

type Balance struct {
	Crypto string  `json:"crypto"`
	Amount float64 `json:"amount"`
}

func NewWsHub(logger zerolog.Logger) *WsHub {
	hub := &WsHub{
		Clients:    make(map[string]map[*websocket.Conn]bool),
		Broadcast:  make(chan WsMessage, 100),
		Register:   make(chan *WsClient, 100),
		Unregister: make(chan *WsClient, 100),
		Logger:     logger,
	}
	return hub
}

func (h *WsHub) Run() {
	for {
		select {
		case client := <-h.Register:
			if h.Clients[client.UserID] == nil {
				h.Clients[client.UserID] = make(map[*websocket.Conn]bool)
			}
			h.Clients[client.UserID][client.Conn] = true
			h.Logger.Info().
				Str("user_id", client.UserID).
				Int("connection_count", len(h.Clients[client.UserID])).
				Msg("WebSocket client registered successfully")

		case client := <-h.Unregister:
			if clients, ok := h.Clients[client.UserID]; ok {
				delete(clients, client.Conn)
				h.Logger.Info().
					Str("user_id", client.UserID).
					Int("connection_count", len(clients)).
					Msg("WebSocket client unregistered")
				if len(clients) == 0 {
					delete(h.Clients, client.UserID)
				}
				client.Conn.Close()
			}

		case message := <-h.Broadcast:
			var userID string
			var logID string
			var logType string

			switch message.Type {
			case "deposit":
				if message.Deposit != nil {
					userID = message.Deposit.UserID
					logID = message.Deposit.SessionID
					logType = "deposit_session"
				}
			case "withdrawal":
				if message.Withdrawal != nil {
					userID = message.Withdrawal.UserID
					logID = message.Withdrawal.WithdrawalID
					logType = "withdrawal"
				}
			case "balance":
				if message.Balance != nil {
					userID = message.Balance.UserID
					logID = message.Balance.CurrencyCode
					logType = "balance_update"
				}
			}

			h.Logger.Info().
				Str("user_id", userID).
				Str("log_id", logID).
				Str("type", message.Type).
				Msg("Broadcasting " + logType + " update")

			if clients, ok := h.Clients[userID]; ok && userID != "" {
				h.Logger.Info().
					Str("user_id", userID).
					Int("client_count", len(clients)).
					Msg("Found clients for broadcast")
				for conn := range clients {
					err := conn.WriteJSON(message)
					if err != nil {
						h.Logger.Err(err).
							Str("user_id", userID).
							Str("log_id", logID).
							Str("type", message.Type).
							Msg("Failed to send WebSocket message")
						conn.Close()
						delete(clients, conn)
					} else {
						h.Logger.Info().
							Str("user_id", userID).
							Str("log_id", logID).
							Str("type", message.Type).
							Msg("WebSocket message sent successfully")
					}
				}
				if len(clients) == 0 {
					delete(h.Clients, userID)
				}
			} else {
				if userID == "" && message.Type == "balance" {
					for userID, clients := range h.Clients {
						for conn := range clients {
							err := conn.WriteJSON(message)
							if err != nil {
								h.Logger.Err(err).
									Str("user_id", userID).
									Str("log_id", logID).
									Str("type", message.Type).
									Msg("Failed to send WebSocket balance message")
								conn.Close()
								delete(clients, conn)
							} else {
								h.Logger.Info().
									Str("user_id", userID).
									Str("log_id", logID).
									Str("type", message.Type).
									Msg("WebSocket balance message sent successfully")
							}
						}
						if len(clients) == 0 {
							delete(h.Clients, userID)
						}
					}
				} else {
					h.Logger.Warn().
						Str("user_id", userID).
						Str("log_id", logID).
						Str("type", message.Type).
						Msg("No clients found for broadcast")
				}
			}
		}
	}
}

func (h *WsHub) BroadcastDepositSession(session domain.DepositSession) {
	h.Logger.Info().
		Str("session_id", session.SessionID).
		Str("user_id", session.UserID).
		Msg("Preparing to broadcast deposit session update")
	h.Broadcast <- WsMessage{
		Type:    "deposit",
		Deposit: &session,
	}
}

func (h *WsHub) BroadcastWithdrawal(withdrawal domain.Withdrawal) {
	h.Logger.Info().
		Str("withdrawal_id", withdrawal.WithdrawalID).
		Str("user_id", withdrawal.UserID).
		Msg("Preparing to broadcast withdrawal update")
	h.Broadcast <- WsMessage{
		Type:       "withdrawal",
		Withdrawal: &withdrawal,
	}
}

func (h *WsHub) BroadcastBalance(balance domain.Balance) {
	h.Logger.Info().
		Str("crypto", balance.CurrencyCode).
		Float64("amount", float64(balance.AmountCents)).
		Msg("Preparing to broadcast balance update")
	h.Broadcast <- WsMessage{
		Type:    "balance",
		Balance: &balance,
	}
}
