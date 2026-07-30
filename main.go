package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

type result string

const (
	inProgress result = "in_progress"
	xWins      result = "x_wins"
	oWins      result = "o_wins"
	draw       result = "draw"
)

var winningLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
	{0, 4, 8}, {2, 4, 6}, // diagonals
}

type GameState struct {
	turn  string
	board [9]string
	mu    sync.Mutex
}

func NewGameState() *GameState {
	return &GameState{turn: "X"}
}

func (game *GameState) computeResult() result {
	for _, line := range winningLines {
		a, b, c := line[0], line[1], line[2]
		if game.board[a] != "" && game.board[a] == game.board[b] && game.board[b] == game.board[c] {
			if game.board[a] == "X" {
				return xWins
			}
			return oWins
		}
	}

	for _, cell := range game.board {
		if cell == "" {
			return inProgress
		}
	}

	return draw
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", relayHandler)

	serverAddress := ":8080"

	fmt.Printf("Starting server on port %s\n", serverAddress)

	err := http.ListenAndServe(serverAddress, mux)
	if err != nil {
		log.Fatalf("Server failed to start %v", err)
	}

}

func relayHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to Relay!")

}
