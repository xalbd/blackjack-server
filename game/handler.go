package game

import (
	"encoding/json"

	"github.com/google/uuid"
)

type clientCommand struct {
	Action string
	Bet    int
	Hand   int
}

func (table *Table) HandleCommand(uuid uuid.UUID, recv []byte) {
	var cmd clientCommand
	err := json.Unmarshal(recv, &cmd)
	if err != nil {
		return
	}

	switch table.status {
	case Betting:
		switch cmd.Action {
		case "bet": // add a bet
			if p := table.playerWithUUID(uuid); p == nil {
				table.Players = append(table.Players, Player{Id: uuid, Money: 1000})
			}
			table.enterBet(uuid, cmd.Bet)
		case "end": // end betting
			table.playerWithUUID(uuid).DoneBetting = true
			if table.allBetsIn() {
				table.dealAll()
				if table.dealer.hasBlackjack() {
					table.status = DealerTurn
				} else {
					table.status = PlayerTurn
				}
			}
		}
	case PlayerTurn: // player turn
		player := table.playerWithUUID(uuid)
		if player == nil || table.Hands[table.ActiveHand].PlayerId != uuid {
			return
		}

		var end bool
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

		if table.bust() || end || table.Hands[table.ActiveHand].bestScore() == 21 {
			table.ActiveHand++
		}

		if table.ActiveHand >= len(table.Hands) {
			table.status = DealerTurn
		}
	}
}
