package main

import (
	"errors"
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
	Board   [9]string `json:"board"`
	Turn    Role      `json:"turn"`
	Result  result    `json:"result"`
	Message string    `json:"message"`
}

func NewGameState() *GameState {
	return &GameState{Turn: xRole}
}

func (game *GameState) checkWin() result {
	for _, line := range winningLines {

		a, b, c := line[0], line[1], line[2]
		if game.Board[a] != "" && game.Board[a] == game.Board[b] && game.Board[b] == game.Board[c] {
			if game.Board[a] == string(xRole) {
				return xWins
			}
			return oWins
		}
	}

	for _, cell := range game.Board {
		if cell == "" {
			return inProgress
		}
	}

	return draw
}

func (game *GameState) MakeMove(position int) error {
	if position < 0 || position > 8 {
		return ErrOutOfRange
	}
	if game.Board[position] != "" {
		return ErrCellOccupied
	}
	if game.checkWin() != inProgress {
		return ErrGameOver
	}

	game.Board[position] = string(game.Turn)

	game.Result = game.checkWin()

	if game.Turn == xRole {
		game.Turn = oRole
	} else {
		game.Turn = xRole
	}

	return nil
}

func (game *GameState) Reset() {
	game.Board = [9]string{}
	game.Result = inProgress
	game.Turn = xRole
}
