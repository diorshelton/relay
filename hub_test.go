package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestApplyMove(t *testing.T) {
	tests := []struct {
		name          string
		initialBoard  [9]string
		movePosition  int
		expectErr     bool
		wantBoardCell string
	}{
		{
			name:          "Valid empty cell move",
			initialBoard:  [9]string{},
			movePosition:  1,
			expectErr:     false,
			wantBoardCell: "X",
		},
		{
			name:          "Invalid occupied cell move",
			initialBoard:  [9]string{"O"},
			movePosition:  0,
			expectErr:     true,
			wantBoardCell: "O",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			game := &GameState{
				Board: tc.initialBoard,
				Turn:  xRole,
			}

			hub := NewHub(game)
			player := &Player{conn: nil, role: xRole}

			err := hub.applyMove(player, tc.movePosition)

			if (err != nil) != tc.expectErr {
				t.Fatalf("applyMove() unexpected error state: %v", err)
			}

			actualCell := game.Board[tc.movePosition]

			if actualCell != tc.wantBoardCell {
				t.Errorf("Board cell state mismatch at position %d: got %q, want %q", tc.movePosition, actualCell, tc.wantBoardCell)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		name         string
		expectErr    bool
		expectedRole Role
		expectedConn int
		conn         *websocket.Conn
	}{
		{
			name:         "Connection one",
			expectErr:    false,
			expectedRole: xRole,
			expectedConn: 1,
			conn:         &websocket.Conn{},
		},
		{
			name:         "Connection two",
			expectErr:    false,
			expectedRole: oRole,
			expectedConn: 2,
			conn:         &websocket.Conn{},
		},
		{
			name:         "Connection three",
			expectErr:    true,
			expectedConn: 2,
			conn:         &websocket.Conn{},
		},
	}

	game := &GameState{
		Turn: xRole,
	}

	hub := &Hub{
		connections: make(map[*websocket.Conn]*Player), room: game,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			player, err := hub.Join(tc.conn)

			conns := len(hub.connections)

			if (err != nil) != tc.expectErr {
				t.Fatalf("Join() unexpected error:%v, player:%+v", err, player)
			}

			if err == nil && tc.expectedRole != player.role {
				t.Fatalf("Expected role %v, got %v", tc.expectedRole, player.role)
			}

			if tc.expectedConn != conns {
				t.Fatalf("hub has %v connections, expected %v connections", conns, tc.expectedConn)
			}
		})
	}
}

func TestBroadcastCount(t *testing.T) {
	t.Parallel()

	game := NewGameState()
	hub := NewHub(game)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusInternalError, "internal error")

		_, err = hub.Join(c)
		if err != nil {
			c.Close(websocket.StatusPolicyViolation, "game already full")
			return
		}

		hub.broadcastCount()

		defer func() {
			hub.Remove(c)
			c.Close(websocket.StatusNormalClosure, "connection closed")
		}()

	}))
	defer s.Close()

	// Dial test server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	wsURL := "ws" + s.URL[len("http"):]
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer c.Close(websocket.StatusGoingAway, "client closing")

	var inMsg ConnectionMessage

	err = wsjson.Read(ctx, c, &inMsg)
	if err != nil {
		t.Fatalf("failed to read %v", err)
	}

	t.Log(inMsg)
}
