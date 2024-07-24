package game

import "testing"

func TestNewDeck(t *testing.T) {
	deck := NewDeck()
	seen := make(map[Card]bool)

	if len(deck) != 52 {
		t.Fatalf("NewDeck() returned a deck with %d cards, expected 52", len(deck))
	}

	for _, card := range deck {
		if seen[card] {
			t.Fatalf("NewDeck() returned a deck with duplicate card %v", card)
		}
		seen[card] = true
	}
}

func TestShuffle(t *testing.T) {
	deck := NewDeck()
	shuffled := NewDeck()
	shuffled.Shuffle()
	seen := make(map[Card]uint8)

	if len(deck) != len(shuffled) {
		t.Fatalf("Shuffle() returned a deck with %d cards, expected %d", len(shuffled), len(deck))
	}

	for _, card := range deck {
		seen[card]++
	}

	for _, card := range shuffled {
		seen[card]++
	}

	for k, v := range seen {
		if v != 2 {
			t.Fatalf("Shuffle() did not preserve card %v", k)
		}
	}
}

func TestDeal(t *testing.T) {
	deck := Deck{{Diamond, Three}, {Club, Four}}

	dealt, ok := deck.Deal()
	if (!ok || dealt != Card{Club, Four} || len(deck) != 1 || deck[0] != Card{Diamond, Three}) {
		t.Fatalf("Deal() returned %v, expected %v", dealt, Card{Spade, Ace})
	}

	dealt, ok = deck.Deal()
	if (!ok || dealt != Card{Diamond, Three} || len(deck) != 0) {
		t.Fatalf("Deal() returned %v, expected %v", dealt, Card{Club, Four})
	}

	dealt, ok = deck.Deal()
	if ok {
		t.Fatalf("Deal() returned %v, expected false", dealt)
	}
}
