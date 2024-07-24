package game

import (
	"bufio"
	"fmt"
	"os"
	"slices"
)

type Player struct {
	hands []Hand
	money int
}

type Table struct {
	deck    Deck
	dealer  Hand
	players []Player
}

func NewTable(p int) Table {
	deck := NewDeck()
	deck.Shuffle()

	players := make([]Player, p)
	for i := range players {
		players[i] = Player{hands: []Hand{Hand{}}, money: 0}
	}

	return Table{
		deck:    deck,
		dealer:  Hand{},
		players: players,
	}
}

func (t *Table) InitialDeal() {
	if len(t.deck) < 2*len(t.players) {
		t.deck = NewDeck()
		t.deck.Shuffle()
	}

	for range 2 {
		for i := range t.players {
			Deal(&t.deck, &t.players[i].hands[0])
		}
		Deal(&t.deck, &t.dealer)
	}
}

func (t *Table) Play() {
	t.InitialDeal()
	fmt.Printf("Dealer: %v + Hole\n", t.dealer.upCard())

	// check for dealer blackjack
	up := t.dealer.upCard().Value()
	if (up == 10 || up == 1) && t.dealer.hasBlackjack() {
		fmt.Println("Dealer has blackjack!")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// player turn
	for i := range t.players {
		for j := 0; j < len(t.players[i].hands); j++ {
			current := &t.players[i].hands[j]

			fmt.Printf("Player %v Hand %v: %v\n", i, j, current)
			fmt.Printf("Scores: %v\n", current.getScores())

			// fill in cards (i.e. previously split)
			if len(current.cards) < 2 {
				for range current.cards {
					Deal(&t.deck, current)

					fmt.Printf("Player %v Hand %v: %v\n", i, j, current)
					fmt.Printf("Scores: %v\n", current.getScores())
				}
			}

			// check for bust
			if current.hasBust() {
				fmt.Println("Player busts!")
				continue
			}

			// TODO: insurance

			// check for player blackjack
			if current.hasBlackjack() {
				fmt.Println("Player has blackjack!")
			}

			// get player input
			for bytes, _, _ := reader.ReadLine(); ; bytes, _, _ = reader.ReadLine() {
				if len(bytes) == 0 {
					continue
				}

				end := false

				switch bytes[0] {
				case 'h':
					Hit(&t.deck, current)
				case 'd':
					Double(&t.deck, current)
					end = true
				case 's':
					end = true
				case 'p':
					if current.canSplit() {
						Split(&t.deck, &t.players[i], j)
						Deal(&t.deck, current)
					}
				}

				if end {
					break
				}

				fmt.Printf("Player %v Hand %v: %v\n", i, j, current)
				fmt.Printf("Scores: %v\n", current.getScores())

				if current.hasBust() {
					fmt.Println("Player busts!")
					break
				}
			}
		}
	}

	// dealer turn
	fmt.Printf("Dealer: %v\n", t.dealer)
	fmt.Printf("Scores: %v\n", t.dealer.getScores())
	for !t.dealer.hasBust() && slices.Max(t.dealer.getScores()) < 17 {
		Deal(&t.deck, &t.dealer)

		fmt.Printf("Dealer: %v\n", t.dealer)
		fmt.Printf("Scores: %v\n", t.dealer.getScores())
	}
}

func Hit(d *Deck, h *Hand) {
	Deal(d, h)
}

func Double(d *Deck, h *Hand) {
	Deal(d, h)
}

func Split(d *Deck, p *Player, idx int) {
	oldHand := &p.hands[idx]
	newHand := Hand{cards: []Card{oldHand.cards[1]}, bet: 0}
	oldHand.cards = oldHand.cards[:1]

	p.hands = slices.Insert(p.hands, idx+1, newHand)
}
