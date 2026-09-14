package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func main() {
	game := NewGameState()
	hub := NewHub(game)

	mux := http.NewServeMux()

	//WebSocket endpoint
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./index.html")
	})

	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {

		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Printf("Handshake failed: %v\n", err)
			return
		}

		log.Println("WebSocket connection established successfully")

		//add client connection to hub
		player, err := hub.Join(c)
		if err != nil {
			hub.Remove(c)
			c.Close(websocket.StatusPolicyViolation, "game already full")
			//TODO: Add fallback for closed connection after removal
			return
		}

		//		hub.broadcastState()
		hub.broadcastCount()

		// Cleanup runs when the user leaves or closes the tab
		defer func() {
			hub.Remove(c)
			c.Close(websocket.StatusNormalClosure, "connection closed")
		}()

		// Keep the connection open and read incoming messages
		ctx := context.Background()
		var msg MoveMessage

		for {
			err := wsjson.Read(ctx, c, &msg)
			if err != nil {
				// Loop breadks immediately if tab closes, triggering defer cleanup
				log.Printf("Read error: %v", err)
				break
			}

			err = player.applyMove(msg.Position)
			if err != nil {
				log.Printf("Move err: %v", err)
			}

		}

	})

	serverAddress := ":8080"

	fmt.Printf("Starting server on port %s\n", serverAddress)

	err := http.ListenAndServe(serverAddress, mux)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}

}
