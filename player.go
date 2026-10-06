package main

import (
	"github.com/coder/websocket"
)

type Player struct {
	conn *websocket.Conn
	role Role
}
