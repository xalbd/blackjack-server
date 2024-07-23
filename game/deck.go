package game

import "math/rand"

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

type (
	Deck []Card
	Hand []Card
)

func NewDeck() Deck {
	var cards Deck
	for s := Spade; s <= Club; s++ {
		for r := Ace; r <= King; r++ {
			cards = append(cards, Card{Suit: s, Rank: r})
		}
	}
	return cards
}

func Shuffle(deck Deck) Deck {
	output := make(Deck, len(deck))
	perm := rand.Perm(len(deck))

	for i, v := range perm {
		output[i] = deck[v]
	}

	return output
}

func (deck *Deck) Deal() (Card, bool) {
	if len(*deck) == 0 {
		return Card{}, false
	}

	card := (*deck)[len(*deck)-1]
	(*deck) = (*deck)[:len(*deck)-1]
	return card, true
}
