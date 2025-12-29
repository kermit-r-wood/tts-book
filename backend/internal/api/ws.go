package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type ProgressMessage struct {
	Type       string `json:"type"` // "progress", "log", "complete"
	Percentage int    `json:"percentage"`
	Message    string `json:"message"`
	ChapterID  string `json:"chapterId"`
}

type Hub struct {
	clients   map[*websocket.Conn]bool
	broadcast chan ProgressMessage
	mu        sync.Mutex
}

var GlobalHub = Hub{
	clients:   make(map[*websocket.Conn]bool),
	broadcast: make(chan ProgressMessage),
}

func (h *Hub) Run() {
	for {
		msg := <-h.broadcast
		h.mu.Lock()
		for client := range h.clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(h.clients, client)
			}
		}
		h.mu.Unlock()
	}
}

func init() {
	go GlobalHub.Run()
}

func WsHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade WS: %v", err)
		return
	}

	GlobalHub.mu.Lock()
	GlobalHub.clients[conn] = true
	GlobalHub.mu.Unlock()

	// Send hello
	conn.WriteJSON(ProgressMessage{Type: "log", Message: "Connected to TTS Backend"})
}

// Helper to broadcast progress
func BroadcastProgress(chapterID string, percent int, msg string) {
	GlobalHub.broadcast <- ProgressMessage{
		Type:       "progress",
		Percentage: percent,
		Message:    msg,
		ChapterID:  chapterID,
	}
}

// Helper to broadcast LLM token
func BroadcastLLMOutput(chapterID string, token string) {
	GlobalHub.broadcast <- ProgressMessage{
		Type:      "llm_output",
		Message:   token,
		ChapterID: chapterID,
	}
}
