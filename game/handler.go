package game

import (
	"encoding/json"
)

type clientCommand struct {
	Action string
	Bet    int64
	Hand   int
}

type MoneyUpdate struct {
	UID   string
	Money int64
}

func (table *Table) HandlePlayerUpdate(uid string, money int64, connect bool) {
	switch connect {
	case true:
		if table.playerWithUID(uid) == nil {
			table.Players = append(table.Players, Player{UID: uid, Money: money, active: true})
		} else {
			table.playerWithUID(uid).active = true
		}
	case false:
		table.playerWithUID(uid).active = false

		// advance play if current player leaves
		if table.status == PlayerTurn && table.currentHand().PlayerUID == uid {
			table.advanceHand()
		}
	}
}

func (table *Table) HandleCommand(uid string, recv []byte) {
	var cmd clientCommand
	err := json.Unmarshal(recv, &cmd)
	if err != nil {
		return
	}

	switch table.status {
	case Betting:
		table.handleBettingCommand(uid, cmd)
	case PlayerTurn:
		table.handlePlayerCommand(uid, cmd)
	}
}

func (table *Table) handleBettingCommand(uid string, cmd clientCommand) {
	switch cmd.Action {
	case "bet":
		table.enterBet(uid, cmd.Bet)
	case "end":
		if len(table.Hands) == 0 {
			break
		}
		table.playerWithUID(uid).DoneBetting = true
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

func (table *Table) handlePlayerCommand(uid string, cmd clientCommand) {
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
		if table.bust() || table.blackjack() || !table.playerWithUID(table.Hands[table.ActiveHand].PlayerUID).active {
			table.advanceHand()
		}
	}
}
