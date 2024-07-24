package game

import "testing"

func TestNewDeck(t *testing.T) {
	deck := makeDeck(1)
	seen := make(map[Card]bool)

	if len(deck.cards) != 52 {
		t.Fatalf("NewDeck() returned a deck with %d cards, expected 52", len(deck.cards))
	}

	for _, card := range deck.cards {
		if seen[card] {
			t.Fatalf("NewDeck() returned a deck with duplicate card %v", card)
		}
		seen[card] = true
	}
}

func TestShuffle(t *testing.T) {
	deck := makeDeck(1)
	shuffled := makeDeck(1)
	shuffled.shuffle()
	seen := make(map[Card]uint8)

	if len(deck.cards) != len(shuffled.cards) {
		t.Fatalf("Shuffle() returned a deck with %d cards, expected %d", len(shuffled.cards), len(deck.cards))
	}

	for _, card := range deck.cards {
		seen[card]++
	}

	for _, card := range shuffled.cards {
		seen[card]++
	}

	for k, v := range seen {
		if v != 2 {
			t.Fatalf("Shuffle() did not preserve card %v", k)
		}
	}
}

func TestDeal(t *testing.T) {
	deck := Deck{cards: []Card{{Diamond, Three}, {Club, Four}}}

	dealt := deck.deal()
	if (dealt != Card{Diamond, Three}) {
		t.Fatalf("Deal() returned %v, expected %v", dealt, Card{Spade, Ace})
	}

	dealt = deck.deal()
	if (dealt != Card{Club, Four}) {
		t.Fatalf("Deal() returned %v, expected %v", dealt, Card{Club, Four})
	}
}
