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

func (table *Table) HandlePlayerUpdate(uuid uuid.UUID, connect bool) {
	switch connect {
	case true:
		if table.playerWithUUID(uuid) == nil {
			table.Players = append(table.Players, Player{Id: uuid, Money: 1000, active: true})
		}
	case false:
		table.playerWithUUID(uuid).active = false

		// advance play if current player leaves
		if table.status == PlayerTurn && table.currentHand().PlayerId == uuid {
			table.advanceHand()
		}
	}
}

func (table *Table) HandleCommand(uuid uuid.UUID, recv []byte) {
	var cmd clientCommand
	err := json.Unmarshal(recv, &cmd)
	if err != nil {
		return
	}

	switch table.status {
	case Betting:
		table.handleBettingCommand(uuid, cmd)
	case PlayerTurn:
		table.handlePlayerCommand(uuid, cmd)
	}
}

func (table *Table) handleBettingCommand(uuid uuid.UUID, cmd clientCommand) {
	switch cmd.Action {
	case "bet":
		table.enterBet(uuid, cmd.Bet)
	case "end":
		if len(table.Hands) == 0 {
			break
		}
		table.playerWithUUID(uuid).DoneBetting = true
		if table.allBetsIn() {
			table.dealAll()
			if table.dealer.hasBlackjack() {
				table.dealerTurn()
			} else {
				table.status = PlayerTurn
				table.advanceHand()
			}
		}
	}
}

func (table *Table) handlePlayerCommand(uuid uuid.UUID, cmd clientCommand) {
	player := table.playerWithUUID(uuid)
	if player == nil || table.Hands[table.ActiveHand].PlayerId != uuid {
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

	if end || table.bust() || table.Hands[table.ActiveHand].bestScore() == 21 {
		table.advanceHand()
	}
}

func (table *Table) advanceHand() {
	table.ActiveHand++

	if table.ActiveHand >= len(table.Hands) {
		table.dealerTurn()
	} else {
		// check for bust/blackjack and then skip for inactive players
		if table.bust() || table.blackjack() || !table.playerWithUUID(table.Hands[table.ActiveHand].PlayerId).active {
			table.advanceHand()
		}
	}
}
