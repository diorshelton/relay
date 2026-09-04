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

	hub := &Hub{
		connections: make(map[*websocket.Conn]*Player),
	}

	game := &GameState{
		turn: xRole,
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			player, err := hub.Join(tc.conn, game)

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

	hub := NewHub()
	game := NewGameState()

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: []string{"*"},
		})
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusInternalError, "internal error")

		_, err = hub.Join(c, game)
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
