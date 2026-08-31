package main

import (
	"testing"
)

func TestApplyMove(t *testing.T) {
	tests := []struct {
		name          string
		initialBoard  [9]string
		movePosition  int
		expectErr     bool
		wantBoardCell string
	}{
		{
			name:          "Valid empty cell move",
			initialBoard:  [9]string{},
			movePosition:  1,
			expectErr:     false,
			wantBoardCell: "X",
		},
		{
			name:          "Invalid occupied cell move",
			initialBoard:  [9]string{"O"},
			movePosition:  0,
			expectErr:     true,
			wantBoardCell: "O",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			game := &GameState{
				board: tc.initialBoard,
				turn:  xRole,
			}

			Player1 := &Player{
				game:   game,
				conn:   nil,
				role:   xRole,
				outbox: make(chan StateResponse, 1),
				done:   make(chan struct{}),
			}

			err := Player1.applyMove(tc.movePosition)

			if (err != nil) != tc.expectErr {
				t.Fatalf("applyMove() unexpected error state: %v", err)

			}

			actualCell := Player1.game.board[tc.movePosition]

			if actualCell != tc.wantBoardCell {
				t.Errorf("Board cell state mismatch at position %d: got %q, want %q", tc.movePosition, actualCell, tc.wantBoardCell)
			}
		})

	}

}
