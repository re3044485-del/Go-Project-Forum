package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"app/internal/domain"
	"app/internal/middleware"
	"app/internal/repository"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan domain.ChatMessage
	UserID int64
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan domain.ChatMessage
	register   chan *Client
	unregister chan *Client
	repo       *repository.PostgresRepository
	mu         sync.RWMutex
}

func NewHub(repo *repository.PostgresRepository) *Hub {
	return &Hub{
		broadcast:  make(chan domain.ChatMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		repo:       repo,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, messageBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg domain.ChatMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf("Invalid message payload: %v", err)
			continue
		}

		msg.SenderID = c.UserID

		// Сохраняем сообщение в БД
		if err := c.Hub.repo.SaveChatMessage(context.Background(), &msg); err != nil {
			log.Printf("Error saving chat message: %v", err)
			continue
		}

		// Транслируем всем подключенным клиентам
		c.Hub.broadcast <- msg
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()

	for msg := range c.Send {
		if err := c.Conn.WriteJSON(msg); err != nil {
			break
		}
	}
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized WS Connection", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan domain.ChatMessage, 256),
		UserID: userID,
	}

	client.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}
