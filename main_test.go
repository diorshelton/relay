package main

import (
	"errors"
	"testing"
)

func TestCheckWin(t *testing.T) {
	tests := []struct {
		name  string
		board [9]string
		want  result
	}{
		{
			name:  "empty board",
			board: [9]string{},
			want:  inProgress,
		},
		{
			name:  "partial board no winner",
			board: [9]string{"X", "O", "", "", "X", "", "", "", ""},
			want:  inProgress,
		},
		{
			name:  "row win for X",
			board: [9]string{"X", "X", "X", "O", "O", "", "", "", ""},
			want:  xWins,
		},
		{
			name:  "column win for O",
			board: [9]string{"O", "", "", "O", "X", "", "O", "X", ""},
			want:  oWins,
		},
		{
			name:  "diagonal win for X",
			board: [9]string{"X", "O", "O", "O", "X", "O", "O", "O", "X"},
			want:  xWins,
		},
		{
			name:  "anti-diagonal win for O",
			board: [9]string{"X", "X", "O", "X", "O", "X", "O", "X", "X"},
			want:  oWins,
		},
		{
			name:  "full board no winner is a draw",
			board: [9]string{"X", "O", "X", "X", "O", "O", "O", "X", "X"},
			want:  draw,
		},
		{
			name:  "win completed on the final move is a win, not a draw",
			board: [9]string{"X", "X", "X", "O", "O", "X", "X", "O", "O"},
			want:  xWins,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			game := &GameState{Board: tt.board}
			if got := game.checkWin(); got != tt.want {
				t.Errorf("computeResult() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMakeMove(t *testing.T) {
	t.Run("valid move updates board and flips turn", func(t *testing.T) {
		game := NewGameState()
		if err := game.MakeMove(4); err != nil {
			t.Fatalf("MakeMove(4) returned error: %v", err)
		}
		if game.Board[4] != "X" {
			t.Errorf("board[4] = %q, want %q", game.Board[4], "X")
		}
		if game.Turn != oRole {
			t.Errorf("turn = %q, want %q", game.Turn, oRole)
		}
	})

	t.Run("negative position is out of range", func(t *testing.T) {
		game := NewGameState()
		if err := game.MakeMove(-1); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("MakeMove(-1) error = %v, want %v", err, ErrOutOfRange)
		}
	})

	t.Run("position above 8 is out of range", func(t *testing.T) {
		game := NewGameState()
		if err := game.MakeMove(9); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("MakeMove(9) error = %v, want %v", err, ErrOutOfRange)
		}
	})

	t.Run("occupied cell is rejected", func(t *testing.T) {
		game := NewGameState()
		if err := game.MakeMove(0); err != nil {
			t.Fatalf("first MakeMove(0) returned error: %v", err)
		}
		if err := game.MakeMove(0); !errors.Is(err, ErrCellOccupied) {
			t.Errorf("MakeMove(0) error = %v, want %v", err, ErrCellOccupied)
		}
	})

	t.Run("move after game over is rejected", func(t *testing.T) {
		game := NewGameState()
		game.Board = [9]string{"X", "X", "X", "O", "O", "", "", "", ""}
		if err := game.MakeMove(5); !errors.Is(err, ErrGameOver) {
			t.Errorf("MakeMove(5) error = %v, want %v", err, ErrGameOver)
		}
	})

	t.Run("turn alternates across moves", func(t *testing.T) {
		game := NewGameState()
		positions := []int{0, 1, 2, 3}
		want := []string{"X", "O", "X", "O"}
		for i, pos := range positions {
			if err := game.MakeMove(pos); err != nil {
				t.Fatalf("MakeMove(%d) returned error: %v", pos, err)
			}
			if game.Board[pos] != want[i] {
				t.Errorf("board[%d] = %q, want %q", pos, game.Board[pos], want[i])
			}
		}
	})
}
func TestReset(t *testing.T) {
	game := NewGameState()
	game.Board = [9]string{"X", "O", "", "", "", "", "", "", ""}
	game.Turn = oRole

	game.Reset()

	if game.Board != ([9]string{}) {
		t.Errorf("board = %v, want empty", game.Board)
	}
	if game.Turn != xRole {
		t.Errorf("turn = %q, want %q", game.Turn, xRole)
	}
}
