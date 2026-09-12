package main

import (
	"errors"

	"github.com/coder/websocket"
)

var (
	ErrWrongTurn = errors.New("opposing player's turn")
)

type Player struct {
	game *GameState
	conn *websocket.Conn
	role Role
}

func (p *Player) applyMove(position int) error {
	//validate move
	if p.role != p.game.Turn {
		return ErrWrongTurn
	}

	return p.game.MakeMove(position)
}
