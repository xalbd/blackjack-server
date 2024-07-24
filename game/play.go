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
	deck.shuffle()

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

func (t *Table) takeBets() {
	t.dealer = Hand{}
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
			t.players[i].hands = append(t.players[i].hands, Hand{bet: bet, active: true})
		}
	}
}

func (t *Table) dealAll() {
	totalHands := 1
	for i := range t.players {
		totalHands += len(t.players[i].hands)
	}

	for range 2 {
		for i := range t.players {
			for j := range t.players[i].hands {
				t.deck.dealTo(&t.players[i].hands[j])
			}
		}
		t.deck.dealTo(&t.dealer)
	}
}

func (t *Table) PlayRound() {
	t.takeBets()
	t.dealAll()
	fmt.Printf("Dealer: %v + Hole\n", t.dealer.cards[1])

	// check for dealer blackjack
	if t.dealer.hasBlackjack() {
		fmt.Println("Dealer has blackjack!")
		t.dealerTurn()
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// player turn
	for i := range t.players {
		player := &t.players[i]
		for j := 0; j < len(player.hands); j++ {
			current := &player.hands[j]
			printHand(i, j, current)

			// fill in cards (i.e. previously split)
			for range 2 - len(current.cards) {
				t.deck.dealTo(current)
				printHand(i, j, current)
			}

			// check for bust
			if bust(current) {
				continue
			}

			// player action if not blackjack
			if !blackjack(current, player) {
				// get player input
				for bytes, _, _ := reader.ReadLine(); ; bytes, _, _ = reader.ReadLine() {
					if len(bytes) == 0 {
						continue
					}

					end := false

					switch bytes[0] {
					case 'h':
						hit(current, &t.deck)
					case 'd':
						end = double(current, player, &t.deck)
					case 's':
						end = true
					case 'p':
						split(j, player, &t.deck)
					}

					if bust(current) {
						break
					} else if current.bestScore() == 21 {
						fmt.Println("Player has 21!")
						break
					} else if end {
						break
					}

					printHand(i, j, current)
				}
			}
		}
	}

	t.dealerTurn()
}

func (t *Table) dealerTurn() {
	fmt.Printf("Dealer: %v\n", t.dealer)
	fmt.Printf("Scores: %v\n", t.dealer.scores())
	for !t.dealer.hasBust() && t.dealer.bestScore() < 17 {
		t.deck.dealTo(&t.dealer)

		fmt.Printf("Dealer: %v\n", t.dealer)
		fmt.Printf("Scores: %v\n", t.dealer.scores())
	}

	if t.dealer.hasBust() {
		fmt.Println("Dealer busts!")
		for i := range t.players {
			player := &t.players[i]
			for j := range t.players[i].hands {
				current := &player.hands[j]

				if current.active && !current.hasBust() {
					player.money += 2 * current.bet
				}
			}
		}
	} else {
		dealerScore := t.dealer.bestScore()
		for i := range t.players {
			player := &t.players[i]
			for j := range t.players[i].hands {
				current := &player.hands[j]

				if !current.active {
					continue
				}

				switch playerScore := current.bestScore(); {
				case playerScore > dealerScore:
					fmt.Printf("Player %v Hand %v wins!\n", i, j)
					player.money += 2 * current.bet
				case playerScore == dealerScore:
					fmt.Printf("Player %v Hand %v pushes!\n", i, j)
					player.money += current.bet
				default:
					fmt.Printf("Player %v Hand %v loses!\n", i, j)
				}
			}
		}
	}
}

func printHand(playerIdx int, handIdx int, h *Hand) {
	fmt.Printf("Player %v Hand %v: %v\n", playerIdx+1, handIdx+1, h)
	fmt.Printf("Scores: %v\n", h.scores())
}

// deals a card to a hand (note does not check for bust, etc)
func hit(hand *Hand, deck *Deck) {
	deck.dealTo(hand)
}

// returns whether player has enough money to double a given hand
func (p Player) canDouble(hand *Hand) bool {
	return p.money >= hand.bet
}

// attempts to double a hand, returns whether double was successful
func double(hand *Hand, player *Player, deck *Deck) bool {
	if player.canDouble(hand) {
		deck.dealTo(hand)
		player.money -= hand.bet
		hand.bet *= 2
		return true
	}
	return false
}

// attempts to split a hand, returns whether split was successful
func split(handIndex int, player *Player, deck *Deck) bool {
	oldHand := &player.hands[handIndex]

	if oldHand.canSplit() && player.canDouble(oldHand) {
		newHand := Hand{cards: []Card{oldHand.cards[1]}, bet: oldHand.bet, active: true}
		oldHand.cards = oldHand.cards[:1]
		deck.dealTo(oldHand)

		player.hands = slices.Insert(player.hands, handIndex+1, newHand)
		return true
	}
	return false
}

// checks if a hand has bust and takes money/sets hand inactive if it has
// returns whether bust was detected
func bust(hand *Hand) bool {
	if hand.hasBust() {
		fmt.Println("Player has bust!")
		hand.bet = 0
		hand.active = false
		return true
	}
	return false
}

// checks if a hand has blackjack and pays out/sets hand inactive if it does
// returns whether blackjack was detected
func blackjack(hand *Hand, player *Player) bool {
	if hand.hasBlackjack() {
		fmt.Println("Player has blackjack!")
		player.money += (5 * hand.bet) / 2
		hand.bet = 0
		hand.active = false
		return true
	}
	return false
}
