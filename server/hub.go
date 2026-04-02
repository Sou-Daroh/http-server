package server

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a single connected WebSocket dashboard session.
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	Send chan []byte
}

// Hub manages all active WebSocket clients using Go channels.
// This is the concurrency centerpiece — a single goroutine processes
// all register/unregister/broadcast operations serially via channels,
// eliminating the need for mutexes on the client map.
type Hub struct {
	clients    map[*Client]bool
	Broadcast  chan AttackEvent
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a fresh Hub ready to accept WebSocket connections.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Broadcast:  make(chan AttackEvent, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

// Run is the Hub's main event loop. Launch this as a goroutine.
// It serially processes channel events, making the client map safe
// without any lock contention.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS] Client connected. Total: %d", len(h.clients))

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[WS] Client disconnected. Total: %d", len(h.clients))

		case event := <-h.Broadcast:
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("[WS] Failed to marshal event: %v", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- data:
				default:
					// Client buffer full — disconnect them
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// WritePump drains the Send channel and writes messages to the WebSocket.
// Each client gets its own WritePump goroutine.
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

// ReadPump listens for client-side close frames.
// We don't expect the dashboard to send data, but we need this
// goroutine to detect disconnections cleanly.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
