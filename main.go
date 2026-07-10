package main

import (
	"fmt"
	"log"
	"net/http"
)

func relayHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to Replay!")

}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", relayHandler)

	serverAddress := ":8080"

	fmt.Printf("Starting sever on port %s", serverAddress)

	err := http.ListenAndServe(serverAddress, mux)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}
}
