package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComputeResult(t *testing.T) {
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
			game := &GameState{board: tt.board}
			if got := game.computeResult(); got != tt.want {
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
		if game.board[4] != "X" {
			t.Errorf("board[4] = %q, want %q", game.board[4], "X")
		}
		if game.turn != oRole {
			t.Errorf("turn = %q, want %q", game.turn, oRole)
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
		game.board = [9]string{"X", "X", "X", "O", "O", "", "", "", ""}
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
			if game.board[pos] != want[i] {
				t.Errorf("board[%d] = %q, want %q", pos, game.board[pos], want[i])
			}
		}
	})
}

func TestHandleState(t *testing.T) {
	game := NewGameState()
	req := httptest.NewRequest(http.MethodGet, "/state", nil)
	w := httptest.NewRecorder()

	game.HandleState(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var got StateResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	want := StateResponse{Board: [9]string{}, Turn: "X", Result: inProgress}
	if got != want {
		t.Errorf("response = %+v, want %+v", got, want)
	}
}

func TestHandleMove(t *testing.T) {
	t.Run("valid move returns 200 with updated state", func(t *testing.T) {
		game := NewGameState()
		req := httptest.NewRequest(http.MethodPost, "/move", strings.NewReader(`{"position":4}`))
		w := httptest.NewRecorder()

		game.HandleMove(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var got StateResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.Board[4] != "X" || got.Turn != "O" {
			t.Errorf("response = %+v, want board[4]=X turn=O", got)
		}
	})

	t.Run("malformed JSON returns 400", func(t *testing.T) {
		game := NewGameState()
		req := httptest.NewRequest(http.MethodPost, "/move", strings.NewReader(`not json`))
		w := httptest.NewRecorder()

		game.HandleMove(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
		var got errorResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.Error != "invalid request body" {
			t.Errorf("error = %q, want %q", got.Error, "invalid request body")
		}
	})

	t.Run("occupied cell returns 400", func(t *testing.T) {
		game := NewGameState()
		if err := game.MakeMove(0); err != nil {
			t.Fatalf("setup MakeMove(0) returned error: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/move", strings.NewReader(`{"position":0}`))
		w := httptest.NewRecorder()

		game.HandleMove(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
		var got errorResponse
		json.NewDecoder(w.Body).Decode(&got)
		if got.Error != ErrCellOccupied.Error() {
			t.Errorf("error = %q, want %q", got.Error, ErrCellOccupied.Error())
		}
	})

	t.Run("out of range position returns 400", func(t *testing.T) {
		game := NewGameState()
		req := httptest.NewRequest(http.MethodPost, "/move", strings.NewReader(`{"position":99}`))
		w := httptest.NewRecorder()

		game.HandleMove(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
		var got errorResponse
		json.NewDecoder(w.Body).Decode(&got)
		if got.Error != ErrOutOfRange.Error() {
			t.Errorf("error = %q, want %q", got.Error, ErrOutOfRange.Error())
		}
	})

	t.Run("move after game over returns 400", func(t *testing.T) {
		game := NewGameState()
		game.board = [9]string{"X", "X", "X", "O", "O", "", "", "", ""}
		req := httptest.NewRequest(http.MethodPost, "/move", strings.NewReader(`{"position":5}`))
		w := httptest.NewRecorder()

		game.HandleMove(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
		var got errorResponse
		json.NewDecoder(w.Body).Decode(&got)
		if got.Error != ErrGameOver.Error() {
			t.Errorf("error = %q, want %q", got.Error, ErrGameOver.Error())
		}
	})
}

func TestReset(t *testing.T) {
	game := NewGameState()
	game.board = [9]string{"X", "O", "", "", "", "", "", "", ""}
	game.turn = oRole

	game.Reset()

	if game.board != ([9]string{}) {
		t.Errorf("board = %v, want empty", game.board)
	}
	if game.turn != xRole {
		t.Errorf("turn = %q, want %q", game.turn, xRole)
	}
}

func TestHandleReset(t *testing.T) {
	freshState := StateResponse{Board: [9]string{}, Turn: "X", Result: inProgress}

	t.Run("reset after moves returns fresh state", func(t *testing.T) {
		game := NewGameState()
		if err := game.MakeMove(0); err != nil {
			t.Fatalf("setup MakeMove(0) returned error: %v", err)
		}
		if err := game.MakeMove(3); err != nil {
			t.Fatalf("setup MakeMove(3) returned error: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/reset", nil)
		w := httptest.NewRecorder()
		game.HandleReset(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var got StateResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got != freshState {
			t.Errorf("response = %+v, want %+v", got, freshState)
		}
	})

	t.Run("reset clears a finished game", func(t *testing.T) {
		game := NewGameState()
		game.board = [9]string{"X", "X", "X", "O", "O", "", "", "", ""}

		req := httptest.NewRequest(http.MethodPost, "/reset", nil)
		w := httptest.NewRecorder()
		game.HandleReset(w, req)

		var got StateResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got.Result != inProgress {
			t.Errorf("result = %v, want %v", got.Result, inProgress)
		}
	})

	t.Run("game is playable again after reset", func(t *testing.T) {
		game := NewGameState()
		game.board = [9]string{"X", "X", "X", "O", "O", "", "", "", ""}
		game.Reset()

		if err := game.MakeMove(0); err != nil {
			t.Errorf("MakeMove(0) after reset returned error: %v, want nil", err)
		}
	})

	t.Run("reset on a fresh game is a no-op", func(t *testing.T) {
		game := NewGameState()

		req := httptest.NewRequest(http.MethodPost, "/reset", nil)
		w := httptest.NewRecorder()
		game.HandleReset(w, req)

		var got StateResponse
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got != freshState {
			t.Errorf("response = %+v, want %+v", got, freshState)
		}
	})
}
