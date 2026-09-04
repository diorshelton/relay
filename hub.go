package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/coder/websocket"
)

var (
	ErrGameFull = errors.New("game is already full")
)

type ConnectionMessage struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

type Hub struct {
	mu          sync.Mutex
	connections map[*websocket.Conn]*Player
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[*websocket.Conn]*Player),
	}
}

func (h *Hub) Join(conn *websocket.Conn, game *GameState) (*Player, error) {
	h.mu.Lock()
	count := len(h.connections)

	var role Role

	switch {
	case count == 0:
		role = xRole
	case count == 1:
		role = oRole
	case count >= 2:
		h.mu.Unlock()
		return nil, ErrGameFull
	}

	newPlayer := Player{game: game, conn: conn, role: role}
	h.connections[conn] = &newPlayer

	h.mu.Unlock()

	return &newPlayer, nil
}

func (h *Hub) Remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.connections, conn)
	h.mu.Unlock()

	h.broadcastCount()
}

func (h *Hub) broadcastCount() {
	h.mu.Lock()
	defer h.mu.Unlock()

	msg := ConnectionMessage{
		Type:  "connection_update",
		Count: len(h.connections),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	//Iterate through every active connection and write the message
	for conn := range h.connections {
		err := conn.Write(context.Background(), websocket.MessageText, payload)
		if err != nil {
			log.Printf("Failed writing to connection: %v", err)
		}
	}
}
