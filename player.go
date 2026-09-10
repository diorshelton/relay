package main

import "github.com/coder/websocket"

type Player struct {
	game *GameState
	conn *websocket.Conn
	role Role
}

func (p *Player) applyMove(position int) error {
	return p.game.MakeMove(position)

}
