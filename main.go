package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func main() {
	game := NewGameState()

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
		defer c.Close(websocket.StatusNormalClosure, "closing connection")

		log.Println("WebSocket connection established successfully")

		//Keep connection open for one message
		ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
		defer cancel()

		var v any
		if err := wsjson.Read(ctx, c, &v); err != nil {
			log.Println("Read error:", err)
			return
		}
		log.Printf("Received from browser: %v", v)

	})

	serverAddress := ":8080"

	fmt.Printf("Starting server on port %s\n", serverAddress)

	err := http.ListenAndServe(serverAddress, mux)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}

}
