package game

import (
	"math/rand"
	"slices"
	"strconv"
)

type Suit uint8

const (
	Spade Suit = iota
	Heart
	Diamond
	Club
)

type Rank uint8

const (
	_ Rank = iota
	Ace
	Two
	Three
	Four
	Five
	Six
	Seven
	Eight
	Nine
	Ten
	Jack
	Queen
	King
)

type Card struct {
	Suit `json:"suit"`
	Rank `json:"rank"`
}

type Deck struct {
	cards []Card
	index int
}

type Hand struct {
	Cards     []Card `json:"cards"`
	Bet       int64  `json:"bet"`
	PlayerUID string `json:"playerId"`
}

func makeDeck(decks int) Deck {
	var cards []Card
	for range decks {
		for s := Spade; s <= Club; s++ {
			for r := Ace; r <= King; r++ {
				cards = append(cards, Card{Suit: s, Rank: r})
			}
		}
	}

	return Deck{cards: cards, index: 0}
}

func (d *Deck) shuffle() {
	cards := d.cards
	rand.Shuffle(len(cards), func(i, j int) {
		cards[i], cards[j] = cards[j], cards[i]
	})
}

func (d *Deck) deal() Card {
	if d.index >= len(d.cards) {
		d.shuffle()
		d.index = 0
	}

	card := d.cards[d.index]
	d.index++
	return card
}

func (d *Deck) dealTo(h *Hand) {
	h.Cards = append(h.Cards, d.deal())
}

func (c Card) value() int {
	switch c.Rank {
	case Jack, Queen, King:
		return 10
	default:
		return int(c.Rank)
	}
}

func (c Card) String() string {
	switch c.Rank {
	case Ace:
		return "A"
	case Jack:
		return "J"
	case Queen:
		return "Q"
	case King:
		return "K"
	default:
		return strconv.Itoa(c.value())
	}
}

func (h Hand) String() string {
	output := ""
	for i, c := range h.Cards {
		if i > 0 {
			output += ", "
		}
		output += c.String()
	}
	return output
}

func (h *Hand) scores() []int {
	hardTotal, aceCount := 0, 0

	for _, c := range h.Cards {
		hardTotal += c.value()

		if c.Rank == Ace {
			aceCount++
		}
	}

	scores := []int{}
	if hardTotal <= 21 {
		scores = append(scores, hardTotal)
	}
	if aceCount > 0 && hardTotal+10 <= 21 {
		scores = append(scores, hardTotal+10)
	}

	return scores
}

func (h *Hand) bestScore() int {
	scores := h.scores()
	best := 0
	if len(scores) > 0 {
		best = slices.Max(scores)
	}
	return best
}

func (h *Hand) hasBlackjack() bool {
	return len(h.Cards) == 2 && h.bestScore() == 21
}

func (h *Hand) hasBust() bool {
	return len(h.scores()) == 0
}
