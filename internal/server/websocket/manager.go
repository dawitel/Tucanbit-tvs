package websocket

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/tuncanbit/tvs/internal/domain/interfaces"
	"github.com/tuncanbit/tvs/internal/domain/models"
	"github.com/tuncanbit/tvs/pkg/config"
)

type Manager struct {
	clients   map[string]interfaces.WebSocketClient
	clientsMu sync.RWMutex
	config    config.WebSocketConfig
}

func NewManager(cfg config.WebSocketConfig) interfaces.WebSocketManager {
	manager := &Manager{
		clients: make(map[string]interfaces.WebSocketClient),
		config:  cfg,
	}

	go manager.cleanupInactiveClients()

	return manager
}

// AddClient adds a new WebSocket client
func (m *Manager) AddClient(client interfaces.WebSocketClient) error {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	m.clients[client.GetID()] = client

	log.Info().
		Str("client_id", client.GetID()).
		Int("total_clients", len(m.clients)).
		Msg("WebSocket client added")

	return nil
}

// RemoveClient removes a WebSocket client
func (m *Manager) RemoveClient(clientID string) error {
	m.clientsMu.Lock()
	defer m.clientsMu.Unlock()

	if client, exists := m.clients[clientID]; exists {
		client.Close()
		delete(m.clients, clientID)

		log.Info().
			Str("client_id", clientID).
			Int("total_clients", len(m.clients)).
			Msg("WebSocket client removed")
	}

	return nil
}

// Broadcast sends a message to all connected clients
func (m *Manager) Broadcast(message *models.StatusUpdate) error {
	m.clientsMu.RLock()
	clients := make([]interfaces.WebSocketClient, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	m.clientsMu.RUnlock()

	var wg sync.WaitGroup
	successCount := 0
	failureCount := 0
	var mu sync.Mutex

	for _, client := range clients {
		wg.Add(1)
		go func(c interfaces.WebSocketClient) {
			defer wg.Done()

			if err := c.Send(message); err != nil {
				mu.Lock()
				failureCount++
				mu.Unlock()

				log.Error().
					Err(err).
					Str("client_id", c.GetID()).
					Msg("Failed to send message to WebSocket client")

				// Remove inactive client
				if !c.IsActive() {
					m.RemoveClient(c.GetID())
				}
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(client)
	}

	wg.Wait()

	log.Debug().
		Int("success_count", successCount).
		Int("failure_count", failureCount).
		Int("total_clients", len(clients)).
		Str("message_type", message.Type).
		Msg("Broadcast completed")

	return nil
}

// SendToClient sends a message to a specific client
func (m *Manager) SendToClient(clientID string, message *models.StatusUpdate) error {
	m.clientsMu.RLock()
	client, exists := m.clients[clientID]
	m.clientsMu.RUnlock()

	if !exists {
		return ErrClientNotFound
	}

	if err := client.Send(message); err != nil {
		log.Error().
			Err(err).
			Str("client_id", clientID).
			Msg("Failed to send message to specific WebSocket client")

		// Remove inactive client
		if !client.IsActive() {
			m.RemoveClient(clientID)
		}

		return err
	}

	return nil
}

// GetClientCount returns the number of connected clients
func (m *Manager) GetClientCount() int {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	return len(m.clients)
}

// cleanupInactiveClients periodically removes inactive clients
func (m *Manager) cleanupInactiveClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		m.clientsMu.Lock()
		inactiveClients := make([]string, 0)

		for clientID, client := range m.clients {
			if !client.IsActive() {
				inactiveClients = append(inactiveClients, clientID)
			}
		}

		for _, clientID := range inactiveClients {
			if client, exists := m.clients[clientID]; exists {
				client.Close()
				delete(m.clients, clientID)
			}
		}

		if len(inactiveClients) > 0 {
			log.Info().
				Int("removed_count", len(inactiveClients)).
				Int("active_clients", len(m.clients)).
				Msg("Cleaned up inactive WebSocket clients")
		}

		m.clientsMu.Unlock()
	}
}
