package game

import (
	"encoding/json"
)

type clientCommand struct {
	Action string
	Bet    int64
	Seat   int
}

type MoneyUpdate struct {
	UID   string
	Money int64
}

type Broadcast struct {
	Dealer      []Card      `json:"dealer"`
	Players     []Player    `json:"players"`
	Hands       []Hand      `json:"hands"`
	ActiveHand  int         `json:"activeHand"`
	TableStatus TableStatus `json:"status"`
}

func (table *Table) HandlePlayerUpdate(uid string, money int64, connect bool) {
	switch connect {
	case true:
		if table.playerWithUID(uid) == nil {
			table.Players = append(table.Players, Player{UID: uid, Money: money, active: true})
		} else {
			table.playerWithUID(uid).active = true
		}
		table.broadcast()
	case false:
		table.playerWithUID(uid).active = false

		switch table.status {
		case Betting:
			for i := range table.Hands {
				if table.Hands[i].PlayerUID == uid && table.Hands[i].Bet == 0 {
					table.Hands[i] = Hand{}
				}
			}
			table.broadcast()
		case PlayerTurn:
			if table.currentHand().PlayerUID == uid {
				table.advanceHand()
			}
		}
	}
}

func (table *Table) HandleCommand(uid string, recv []byte) {
	var cmd clientCommand
	err := json.Unmarshal(recv, &cmd)
	if err != nil {
		return
	}

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
		table.handlePlayerCommand(uid, cmd)
	}
}

func (table *Table) handleBettingCommand(uid string, cmd clientCommand) {
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

	table.broadcast()

	if end || table.bust() || table.Hands[table.ActiveHand].bestScore() == 21 {
		table.advanceHand()
	}
}
