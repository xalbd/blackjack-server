package game

import (
	"slices"

	"github.com/google/uuid"
)

type TableStatus int

const (
	Betting TableStatus = iota
	PlayerTurn
)

type Player struct {
	Id          uuid.UUID `json:"id"`
	Money       int       `json:"money"`
	DoneBetting bool      `json:"doneBetting"`
	active      bool
}

type Table struct {
	deck       Deck
	dealer     Hand
	minBet     int
	status     TableStatus
	Players    []Player
	Hands      []Hand
	ActiveHand int
}

func NewTable() Table {
	deck := makeDeck(1)
	deck.shuffle()

	return Table{
		deck:       deck,
		minBet:     10,
		ActiveHand: -1,
	}
}

func (t *Table) ResetHands() {
	t.ActiveHand = -1
	t.status = Betting
	t.dealer = Hand{}
	t.Hands = []Hand{}

	for i := range t.Players {
		t.Players[i].DoneBetting = false
	}
}

func (t *Table) playerWithUUID(uuid uuid.UUID) *Player {
	for i := range t.Players {
		if t.Players[i].Id == uuid {
			return &t.Players[i]
		}
	}
	return nil
}

func (t *Table) currentHand() *Hand {
	return &t.Hands[t.ActiveHand]
}

func (t *Table) enterBet(uuid uuid.UUID, bet int) {
	player := t.playerWithUUID(uuid)

	if player.DoneBetting || bet < t.minBet || bet > player.Money {
		return
	}

	player.Money -= bet
	t.Hands = append(t.Hands, Hand{Bet: bet, PlayerId: player.Id})
}

func (t *Table) dealAll() {
	for range 2 {
		for i := range t.Hands {
			t.deck.dealTo(&t.Hands[i])
		}
		t.deck.dealTo(&t.dealer)
	}
}

func (t *Table) allBetsIn() bool {
	for _, p := range t.Players {
		if !p.DoneBetting && p.active && p.Money > t.minBet {
			return false
		}
	}

	return true
}

func (table *Table) dealerTurn() {
	for !table.dealer.hasBust() && table.dealer.bestScore() < 17 {
		table.deck.dealTo(&table.dealer)
	}

	d := table.dealer.bestScore()
	for i := range table.Hands {
		p := table.playerWithUUID(table.Hands[i].PlayerId)
		h := &table.Hands[i]

		if h.bestScore() > d {
			p.Money += 2 * h.Bet
		} else if h.bestScore() == d {
			p.Money += h.Bet
		}

		h.Bet = 0
	}

	table.ResetHands()
}

// deals a card to the current hand
func (t *Table) hit() {
	t.deck.dealTo(t.currentHand())
}

// returns whether current hand can be doubled
func (t *Table) canDouble() bool {
	hand := t.currentHand()
	player := t.playerWithUUID(hand.PlayerId)
	return player.Money >= hand.Bet
}

// returns whether current hand can be split
func (t *Table) canSplit() bool {
	hand := t.currentHand()
	return t.canDouble() && len(hand.Cards) == 2 && hand.Cards[0].value() == hand.Cards[1].value()
}

// attempts to double current hand, returns whether double was successful
func (t *Table) double() bool {
	hand := t.currentHand()
	player := t.playerWithUUID(hand.PlayerId)

	if t.canDouble() {
		t.hit()
		player.Money -= hand.Bet
		hand.Bet *= 2
		return true
	}
	return false
}

// attempts to split current hand, returns whether split was successful
func (t *Table) split() bool {
	oldHand := t.currentHand()

	if t.canSplit() {
		newHand := Hand{Cards: []Card{oldHand.Cards[1]}, Bet: oldHand.Bet}
		oldHand.Cards = oldHand.Cards[:1]
		t.Hands = slices.Insert(t.Hands, t.ActiveHand+1, newHand)
		return true
	}
	return false
}

// checks if current hand has bust and takes money if it has
// returns whether bust was detected
func (t *Table) bust() bool {
	if t.currentHand().hasBust() {
		t.currentHand().Bet = 0
		return true
	}
	return false
}

// checks if current hand has blackjack and pays out if it does
// returns whether blackjack was detected
func (t *Table) blackjack() bool {
	hand := t.currentHand()
	player := t.playerWithUUID(hand.PlayerId)

	if hand.hasBlackjack() {
		player.Money += (5 * hand.Bet) / 2
		hand.Bet = 0
		return true
	}
	return false
}
