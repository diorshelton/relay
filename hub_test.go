package main

import (
	"testing"

	"github.com/coder/websocket"
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
