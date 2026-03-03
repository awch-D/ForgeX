// Package bus provides the WebSocket delivery mechanism for ForgeX internal events.
package bus

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all cross-origin for local dashboard CLI
	},
}

// Client represents an active WebSocket connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// writePump dumps messages from the hub to the websocket connection.
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

// Hub maintains the set of active clients and broadcasts messages to the clients.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client

	eventBus *protocol.EventBus
}

// NewHub creates a new Hub.
func NewHub(eventBus *protocol.EventBus) *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		eventBus:   eventBus,
	}
}

// Run starts the hub loop for handling client registrations and broadcasts.
func (h *Hub) Run() {
	go h.listenEventBus()
	go h.pollCostUpdates()

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			logger.L().Debugw("🖥 Dashboard connected")
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				logger.L().Debugw("🖥 Dashboard disconnected")
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// listEventBus blocks and reads from the system event bus, formatting for the frontend.
func (h *Hub) listenEventBus() {
	if h.eventBus == nil {
		return
	}

	// Subscribe to all agent events
	ch := h.eventBus.SubscribeAll(100)

	for msg := range ch {
		// Serialize protocol.Message to JSON payload
		payload := struct {
			Topic string           `json:"topic"`
			Data  protocol.Message `json:"data"`
		}{
			Topic: "agent_event",
			Data:  msg,
		}

		b, err := json.Marshal(payload)
		if err == nil {
			h.broadcast <- b
		}
	}
}

// pollCostUpdates occasionally checks the global ledger and pushes updates
func (h *Hub) pollCostUpdates() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var lastTokens int
	for range ticker.C {
		tokens, costUSD := cost.Global().Summary()
		if tokens != lastTokens {
			lastTokens = tokens
			payload := struct {
				Topic  string  `json:"topic"`
				Tokens int     `json:"tokens"`
				Cost   float64 `json:"cost"`
			}{
				Topic:  "sys_cost",
				Tokens: tokens,
				Cost:   costUSD,
			}
			b, _ := json.Marshal(payload)
			h.broadcast <- b
		}
	}
}

// HandleWebSocket upgrades the HTTP connection and registers the client.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L().Errorw("Failed to upgrade websocket", "error", err)
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Start the write pump that sends messages from hub to conn
	go client.writePump()

	// Keep connection alive, draining reads so writer can work
	go func() {
		defer func() {
			client.hub.unregister <- client
			client.conn.Close()
		}()
		for {
			_, _, err := client.conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
