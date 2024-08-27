package game

import (
	"encoding/json"
)

type playerCommand struct {
	Action string
	Bet    int64
	Seat   int
}

// data sent to all players whenever game state changes
type broadcast struct {
	Dealer      []card      `json:"dealer"`
	Players     []player    `json:"players"`
	Hands       []Hand      `json:"hands"`
	ActiveHand  int         `json:"activeHand"`
	TableStatus tableStatus `json:"status"`
}

func (table *table) handlePlayerUpdate(cmd playersUpdate) {
	switch cmd.connect {
	case true:
		if table.playerWithUID(cmd.playerId) == nil {
			table.Players = append(table.Players, player{UID: cmd.playerId, Money: cmd.money, active: true})
		} else {
			table.playerWithUID(cmd.playerId).active = true
		}
		table.broadcast()
	case false:
		table.playerWithUID(cmd.playerId).active = false

		switch table.status {
		case Betting:
			for i := range table.Hands {
				if table.Hands[i].PlayerUID == cmd.playerId && table.Hands[i].Bet == 0 {
					table.Hands[i] = Hand{}
				}
			}
			table.broadcast()
		case PlayerTurn:
			if table.currentHand().PlayerUID == cmd.playerId {
				table.advanceHand()
			}
		}
	}
}

func (table *table) handleCommand(cmd wsCommand) {
	var pc playerCommand
	err := json.Unmarshal(cmd.message, &pc)
	if err != nil {
		return
	}

	switch pc.Action {
	case "join":
		table.join(cmd.playerId, pc.Seat)
	case "leave":
		table.leave(cmd.playerId, pc.Seat)
	}

	switch table.status {
	case Betting:
		table.handleBettingCommand(cmd.playerId, pc)
	case PlayerTurn:
		table.handleActionCommand(cmd.playerId, pc)
	}
}

func (table *table) handleBettingCommand(uid string, cmd playerCommand) {
	switch cmd.Action {
	case "bet":
		table.enterBet(uid, cmd.Bet, cmd.Seat)
	}

	if table.allBetsIn() {
		table.status = PlayerTurn
		table.dealAll()
		if table.dealer.hasBlackjack() {
			table.dealerTurn()
		} else {
			table.advanceHand()
		}
	}
}

func (table *table) handleActionCommand(uid string, cmd playerCommand) {
	player := table.playerWithUID(uid)
	if player == nil || table.Hands[table.ActiveHand].PlayerUID != uid {
		return
	}

	end := false
	switch cmd.Action {
	case "hit":
		table.hit()
	case "stand":
		end = true
	case "double":
		end = table.double()
	case "split":
		table.split()
	}

	table.broadcast()

	if end || table.bust() || table.Hands[table.ActiveHand].bestScore() == 21 {
		table.advanceHand()
	}
}
