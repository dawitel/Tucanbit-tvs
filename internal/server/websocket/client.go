package websocket

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/domain/models"
)

var (
	ErrClientNotFound = errors.New("client not found")
	ErrClientInactive = errors.New("client is inactive")
)

// Client implements the WebSocketClient interface
type Client struct {
	id     string
	conn   *websocket.Conn
	active bool
	send   chan *models.StatusUpdate
	done   chan struct{}
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn) interfaces.WebSocketClient {
	client := &Client{
		id:     uuid.New().String(),
		conn:   conn,
		active: true,
		send:   make(chan *models.StatusUpdate, 256),
		done:   make(chan struct{}),
	}

	// Start message sender and reader goroutines
	go client.writePump()
	go client.readPump()

	return client
}

// GetID returns the client ID
func (c *Client) GetID() string {
	return c.id
}

// Send sends a message to the client
func (c *Client) Send(message *models.StatusUpdate) error {
	if !c.active {
		return ErrClientInactive
	}

	select {
	case c.send <- message:
		return nil
	case <-c.done:
		return ErrClientInactive
	default:
		// Channel is full, drop message to prevent blocking
		log.Warn().Str("client_id", c.id).Msg("WebSocket client send channel full, dropping message")
		return errors.New("send channel full")
	}
}

// Close closes the client connection
func (c *Client) Close() error {
	if !c.active {
		return nil
	}

	c.active = false
	close(c.done)

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// IsActive checks if the client connection is active
func (c *Client) IsActive() bool {
	return c.active
}

// HandleConnection handles the WebSocket connection lifecycle
func (c *Client) HandleConnection() {
	defer c.Close()

	// Wait for connection to be closed
	<-c.done
}

// readPump handles incoming messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case <-c.done:
			return
		default:
			_, _, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Str("client_id", c.id).Msg("Unexpected WebSocket close error")
				}
				return
			}
		}
	}
}

// writePump handles outgoing messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Error().Err(err).Str("client_id", c.id).Msg("Failed to marshal WebSocket message")
				w.Close()
				continue
			}

			w.Write(data)

			// Send any queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				additionalMessage := <-c.send
				additionalData, err := json.Marshal(additionalMessage)
				if err != nil {
					log.Error().Err(err).Str("client_id", c.id).Msg("Failed to marshal additional WebSocket message")
					continue
				}
				w.Write(additionalData)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.done:
			return
		}
	}
}
