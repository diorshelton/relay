package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

var (
	ErrGameFull  = errors.New("game is already full")
	ErrWrongTurn = errors.New("opposing player's turn")
)

type MoveMessage struct {
	Position int `json:"position"`
}

type ErrorMessage struct {
	Error string `json:"error"`
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

	newPlayer := Player{conn: conn, role: role}
	h.connections[conn] = &newPlayer

	return &newPlayer, nil
}

func (h *Hub) applyMove(player *Player, position int) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if player.role != h.room.Turn {
		return ErrWrongTurn
	}

	return h.room.MakeMove(position)
}

func (h *Hub) sendError(conn *websocket.Conn, moveErr error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	msg := ErrorMessage{Error: moveErr.Error()}

	return wsjson.Write(ctx, conn, msg)
}

// EndGame tears down the current round: closes every connection, clears the
// connection map, and resets the room to a fresh game. A single player
// leaving ends the game for whoever's left.
//
// Guarded so a call with nothing left to clean up (a stale call from a
// connection whose read loop only failed because EndGame already closed its
// socket) is a no-op, rather than resetting a room that a later "play again"
// reconnect may have already repopulated.
func (h *Hub) EndGame() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.connections) == 0 {
		return
	}

	for conn := range h.connections {
		conn.Close(websocket.StatusNormalClosure, "opponent left")
		delete(h.connections, conn)
	}

	h.room.Reset()
}

func (h *Hub) broadcastState() {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	for conn := range h.connections {
		err := wsjson.Write(ctx, conn, h.room)
		if err != nil {
			log.Printf("Failed writing to connection: %v", err)
		}
	}
}
