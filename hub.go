package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
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
	room        *GameState
	connections map[*websocket.Conn]*Player
}

func NewHub(game *GameState) *Hub {
	return &Hub{
		connections: make(map[*websocket.Conn]*Player),
		room:        game,
	}
}

func (h *Hub) Join(conn *websocket.Conn) (*Player, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	count := len(h.connections)

	if count >= 2 {
		return nil, ErrGameFull
	}

	role := oRole
	if count == 0 {
		role = xRole
	}

	newPlayer := Player{game: h.room, conn: conn, role: role}
	h.connections[conn] = &newPlayer

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

func (h *Hub) broadcastState() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.connections {
		err := wsjson.Write(context.Background(), conn, h.room)
		if err != nil {
			log.Printf("Failed writing to connection: %v", err)
		}
	}
}
