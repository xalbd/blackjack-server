package game

import (
	"encoding/json"
	"time"
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
	Time        int64       `json:"time"`
}

func (table *table) handlePlayerUpdate(cmd playersUpdate) {
	switch cmd.connect {
	case true:
		if table.playerWithUID(cmd.playerId) == nil {
			table.Players = append(table.Players, player{UID: cmd.playerId, DisplayName: cmd.displayName, active: true})
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

func (table *table) handleWSCommand(cmd wsCommand) {
	var pc playerCommand
	err := json.Unmarshal(cmd.message, &pc)
	if err != nil {
		return
	}

	table.handleCommand(cmd.playerId, pc)
}

func (table *table) handleCommand(uid string, cmd playerCommand) {
	switch cmd.Action {
	case "join":
		table.join(uid, cmd.Seat)
	case "leave":
		table.leave(uid, cmd.Seat)
	}

	switch table.status {
	case Betting:
		table.handleBettingCommand(uid, cmd)
	case PlayerTurn:
		table.handleActionCommand(uid, cmd)
	}
}

func (table *table) handleNullAction() {
	switch table.status {
	case Betting:
		for i := range table.Hands {
			if table.Hands[i].PlayerUID != "" && table.Hands[i].Bet == 0 {
				table.Hands[i].PlayerUID = ""
			}
		}
		table.broadcast()
		table.startPlayerTurn()

	case PlayerTurn:
		table.advanceHand()
	}
}

func (table *table) handleBettingCommand(uid string, cmd playerCommand) {
	switch cmd.Action {
	case "bet":
		table.enterBet(uid, cmd.Bet, cmd.Seat)
	}

	if table.allBetsIn() {
		table.startPlayerTurn()
	}
}

func (table *table) handleActionCommand(uid string, cmd playerCommand) {
	player := table.playerWithUID(uid)
	if player == nil || table.Hands[table.ActiveHand].PlayerUID != uid {
		return
	}

	end, success := false, false
	switch cmd.Action {
	case "hit":
		success = table.hit()
	case "stand":
		end, success = true, true
	case "double":
		end = table.double()
		success = end
	case "split":
		success = table.split()
	}

	if end || table.bust() || table.Hands[table.ActiveHand].bestScore() == 21 {
		table.advanceHand()
	} else if success {
		table.actionTimeStart = time.Now()
		table.broadcast()
	}
}
