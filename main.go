package main

import (
	"github.com/xalbd/blackjack-server/game"
)

func main() {
	table := game.NewTable(2)
	for {
		table.PlayRound()
	}
}
