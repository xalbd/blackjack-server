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
	minBet  int
}

func NewTable(p int) Table {
	deck := makeDeck(1)
	deck.Shuffle()

	players := make([]Player, p)
	for i := range players {
		players[i] = Player{hands: []Hand{}, money: 1000}
	}

	return Table{
		deck:    deck,
		dealer:  Hand{},
		players: players,
		minBet:  10,
	}
}

func (t *Table) TakeBets() {
	for i := range t.players {
		t.players[i].hands = []Hand{}
		for t.players[i].money > t.minBet {
			fmt.Printf("Player %v has $%v. Enter bet amount (min %v) or 0 to end betting\n", i, t.players[i].money, t.minBet)

			var bet int
			fmt.Scan(&bet)

			if bet == 0 {
				break
			}

			if bet < t.minBet || bet > t.players[i].money {
				fmt.Println("Invalid bet.")
				continue
			}

			t.players[i].money -= bet
			t.players[i].hands = append(t.players[i].hands, Hand{bet: bet})
		}
	}
}

func (t *Table) DealAll() {
	totalHands := 1
	for i := range t.players {
		totalHands += len(t.players[i].hands)
	}

	for range 2 {
		for i := range t.players {
			for j := range t.players[i].hands {
				t.deck.DealTo(&t.players[i].hands[j])
			}
		}
		t.deck.DealTo(&t.dealer)
	}
}

func (t *Table) PlayRound() {
	t.TakeBets()
	t.DealAll()
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
			fmt.Printf("Player %v Hand %v: %v\n", i, j, t.players[i].hands[j])
			fmt.Printf("Scores: %v\n", t.players[i].hands[j].getScores())

			// fill in cards (i.e. previously split)
			if len(t.players[i].hands[j].cards) < 2 {
				for range t.players[i].hands[j].cards {
					t.deck.DealTo(&t.players[i].hands[j])

					fmt.Printf("Player %v Hand %v: %v\n", i, j, t.players[i].hands[j])
					fmt.Printf("Scores: %v\n", t.players[i].hands[j].getScores())
				}
			}

			// check for bust
			if t.players[i].hands[j].hasBust() {
				continue
			}

			// TODO: insurance

			// check for player blackjack
			if t.players[i].hands[j].hasBlackjack() {
				fmt.Println("Player has blackjack!")
			} else {
				// get player input
				for bytes, _, _ := reader.ReadLine(); ; bytes, _, _ = reader.ReadLine() {
					if len(bytes) == 0 {
						continue
					}

					end := false

					switch bytes[0] {
					case 'h':
						Hit(&t.deck, &t.players[i].hands[j])
					case 'd':
						if t.players[i].hasEnoughMoneyToDouble(j) {
							Double(&t.deck, &t.players[i], j)
							end = true
						}
					case 's':
						end = true
					case 'p':
						if t.players[i].hands[j].canSplit() && t.players[i].hasEnoughMoneyToDouble(j) {
							Split(&t.deck, &t.players[i], j)
							t.deck.DealTo(&t.players[i].hands[j])
						}
					}

					fmt.Printf("Player %v Hand %v: %v\n", i, j, t.players[i].hands[j])
					fmt.Printf("Scores: %v\n", t.players[i].hands[j].getScores())

					if t.players[i].hands[j].hasBust() {
						fmt.Println("Player has bust!")
						break
					} else if t.players[i].hands[j].hasBlackjack() {
						fmt.Println("Player has blackjack!")
						break
					} else if end {
						break
					}
				}
			}
		}
	}

	t.DealerTurn()
}

func (t *Table) DealerTurn() {
	fmt.Printf("Dealer: %v\n", t.dealer)
	fmt.Printf("Scores: %v\n", t.dealer.getScores())
	for !t.dealer.hasBust() && slices.Max(t.dealer.getScores()) < 17 {
		t.deck.DealTo(&t.dealer)

		fmt.Printf("Dealer: %v\n", t.dealer)
		fmt.Printf("Scores: %v\n", t.dealer.getScores())
	}

	if t.dealer.hasBust() {
		fmt.Println("Dealer busts!")
		for i := range t.players {
			for j := range t.players[i].hands {
				if !t.players[i].hands[j].hasBust() {
					t.players[i].money += 2 * t.players[i].hands[j].bet
				}
			}
		}
	} else {
		dealerScore := slices.Max(t.dealer.getScores())
		for i := range t.players {
			for j := range t.players[i].hands {
				var playerScore int
				if t.players[i].hands[j].hasBust() {
					playerScore = 0
				} else {
					playerScore = slices.Max(t.players[i].hands[j].getScores())
				}

				if playerScore > dealerScore {
					fmt.Printf("Player %v Hand %v wins!\n", i, j)
					t.players[i].money += 2 * t.players[i].hands[j].bet
				} else if playerScore == dealerScore {
					fmt.Printf("Player %v Hand %v pushes!\n", i, j)
					t.players[i].money += t.players[i].hands[j].bet
				} else {
					fmt.Printf("Player %v Hand %v loses!\n", i, j)
				}
			}
		}
	}
}

func Hit(d *Deck, h *Hand) {
	d.DealTo(h)
}

func (p Player) hasEnoughMoneyToDouble(idx int) bool {
	return p.money >= p.hands[idx].bet
}

func Double(d *Deck, p *Player, idx int) {
	d.DealTo(&p.hands[idx])
	p.money -= p.hands[idx].bet
	p.hands[idx].bet *= 2
}

func Split(d *Deck, p *Player, idx int) {
	oldHand := &p.hands[idx]
	newHand := Hand{cards: []Card{oldHand.cards[1]}, bet: 0}
	oldHand.cards = oldHand.cards[:1]

	p.hands = slices.Insert(p.hands, idx+1, newHand)
}
