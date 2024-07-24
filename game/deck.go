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
	Suit
	Rank
}

type Deck []Card

type Hand struct {
	cards []Card
	bet   int
}

func NewDeck() Deck {
	var cards Deck
	for s := Spade; s <= Club; s++ {
		for r := Ace; r <= King; r++ {
			cards = append(cards, Card{Suit: s, Rank: r})
		}
	}
	return cards
}

func (d *Deck) Shuffle() {
	rand.Shuffle(len(*d), func(i, j int) {
		(*d)[i], (*d)[j] = (*d)[j], (*d)[i]
	})
}

func (d *Deck) Deal() (Card, bool) {
	if len(*d) == 0 {
		return Card{}, false
	}

	card := (*d)[len(*d)-1]
	(*d) = (*d)[:len(*d)-1]
	return card, true
}

func Deal(d *Deck, h *Hand) {
	if dealt, ok := d.Deal(); ok {
		h.cards = append(h.cards, dealt)
	}
}

func (c Card) Value() int {
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
	default:
		return strconv.Itoa(c.Value())
	}
}

func (h Hand) String() string {
	output := ""
	for i, c := range h.cards {
		if i > 0 {
			output += ", "
		}
		output += c.String()
	}
	return output
}

func (h *Hand) upCard() Card {
	return h.cards[1]
}

func (h *Hand) getScores() []int {
	hardTotal, aceCount := 0, 0

	for _, c := range h.cards {
		hardTotal += c.Value()

		if c.Rank == Ace {
			aceCount++
		}
	}

	scores := []int{}
	if hardTotal <= 21 {
		scores = append(scores, hardTotal)

		if aceCount > 0 && hardTotal+10 <= 21 {
			scores = append(scores, hardTotal+10)
		}
	}

	return scores
}

func (h *Hand) hasBlackjack() bool {
	return len(h.cards) == 2 && slices.Contains(h.getScores(), 21)
}

func (h *Hand) hasBust() bool {
	return len(h.getScores()) == 0
}

func (h *Hand) canSplit() bool {
	return len(h.cards) == 2 && h.cards[0].Value() == h.cards[1].Value()
}
