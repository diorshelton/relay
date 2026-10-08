package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

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
			// Join never added c to the hub, so there's nothing to tear down —
			// just reject this connection.
			c.Close(websocket.StatusPolicyViolation, "game already full")
			return
		}

		hub.broadcastState()

		// Cleanup runs when the user leaves or closes the tab
		defer hub.EndGame()

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

			err = hub.applyMove(player, msg.Position)
			if err != nil {
				log.Printf("Move err: %v", err)
				if sendErr := hub.sendError(c, err); sendErr != nil {
					log.Printf("Failed writing to connection: %v", sendErr)
				}
			} else {
				hub.broadcastState()
			}
		}
	})

	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}

	addr := ":" + portStr

	fmt.Printf("Starting server on port %s\n", portStr)

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}

}
