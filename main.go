package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

func main() {
	game := NewGameState()
	hub := NewHub()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")

	})

	mux.HandleFunc("GET /state", game.HandleState)
	mux.HandleFunc("POST /move", game.HandleMove)
	mux.HandleFunc("POST /reset", game.HandleReset)

	//Test WebSocket endpoint
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, r *http.Request) {
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
		hub.Add(c)

		// Cleanup runs when the user leaves or closes the tab
		defer func() {
			hub.Remove(c)
			c.Close(websocket.StatusNormalClosure, "connection closed")
		}()

		// Keep the connection open and read incoming messages
		ctx := r.Context()
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				// Loop breadks immediately if tab closes, triggering defer cleanup
				log.Printf("Read error (connection dropped): %v", err)
				break
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
