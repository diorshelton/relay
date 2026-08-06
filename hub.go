package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/coder/websocket"
)

type ConnectionMessage struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type Hub struct {
	mu          sync.Mutex
	connections map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[*websocket.Conn]struct{}),
	}
}

func (h *Hub) Add(conn *websocket.Conn) {
	h.mu.Lock()
	h.connections[conn] = struct{}{}
	count := len(h.connections)
	h.mu.Unlock()

	h.broadcastCount(count)
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.connections, conn)
	count := len(h.connections)
	h.mu.Unlock()

	h.broadcastCount(count)
}

func (h *Hub) broadcastCount(count int) {
	msg := ConnectionMessage{
		Type:  "connection_update",
		Count: count,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	//Iterate through every active connection and write the message
	for conn := range h.connections {
		err := conn.Write(context.Background(), websocket.MessageText, payload)
		if err != nil {
			log.Printf("Failed writing to connection: %v", err)
		}
	}
}

