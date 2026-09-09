package main

import "github.com/coder/websocket"

type Player struct {
	game   *GameState
	conn   *websocket.Conn
	role   Role
	outbox chan StateResponse
	done   chan struct{}
}

func (p *Player) applyMove(position int) error {
	return p.game.MakeMove(position)

}
