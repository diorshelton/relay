package main

import (
	"errors"
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

type Role string

const (
	xRole Role = "X"
	oRole Role = "O"
)

var winningLines = [8][3]int{
	{0, 1, 2}, {3, 4, 5}, {6, 7, 8}, // rows
	{0, 3, 6}, {1, 4, 7}, {2, 5, 8}, // columns
	{0, 4, 8}, {2, 4, 6}, // diagonals
}

type GameState struct {
	turn  Role
	board [9]string
	mu    sync.Mutex
}

func NewGameState() *GameState {
	return &GameState{turn: xRole}
}

func (game *GameState) computeResult() result {
	for _, line := range winningLines {
		a, b, c := line[0], line[1], line[2]
		if game.board[a] != "" && game.board[a] == game.board[b] && game.board[b] == game.board[c] {
			if game.board[a] == string(xRole) {
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

	game.board[position] = string(game.turn)
	if game.turn == xRole {
		game.turn = oRole
	} else {
		game.turn = xRole
	}

	return nil
}

func (game *GameState) Reset() {
	game.mu.Lock()
	defer game.mu.Unlock()

	game.board = [9]string{}
	game.turn = xRole
}

type StateResponse struct {
	Board  [9]string `json:"board"`
	Turn   Role      `json:"turn"`
	Result result    `json:"result"`
}
