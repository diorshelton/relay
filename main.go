package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	ErrOutOfRange   = errors.New("position out of range")
	ErrCellOccupied = errors.New("cell already occupied")
	ErrGameOver     = errors.New("game is already over")
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

func (game *GameState) MakeMove(position int) error {
	game.mu.Lock()
	defer game.mu.Unlock()

	if position < 0 || position > 8 {
		return ErrOutOfRange
	}
	if game.board[position] != "" {
		return ErrCellOccupied
	}
	if game.computeResult() != inProgress {
		return ErrGameOver
	}

	game.board[position] = game.turn
	if game.turn == "X" {
		game.turn = "O"
	} else {
		game.turn = "X"
	}

	return nil
}

type StateResponse struct {
	Board  [9]string `json:"board"`
	Turn   string    `json:"turn"`
	Result result    `json:"result"`
}

type moveRequest struct {
	Position int `json:"position"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (game *GameState) State() StateResponse {
	game.mu.Lock()
	defer game.mu.Unlock()

	return StateResponse{
		Board:  game.board,
		Turn:   game.turn,
		Result: game.computeResult(),
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: msg})
}

func (game *GameState) HandleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game.State())
}

func (game *GameState) HandleMove(w http.ResponseWriter, r *http.Request) {
	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := game.MakeMove(req.Position); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game.State())
}

func main() {
	game := NewGameState()

	mux := http.NewServeMux()

	mux.HandleFunc("/", relayHandler)
	mux.HandleFunc("/state", game.HandleState)
	mux.HandleFunc("/move", game.HandleMove)

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
