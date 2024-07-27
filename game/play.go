package game

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"strconv"
)

type Player struct {
	Hands []Hand
	Money int
}

func (p Player) String() string {
	return "Money: " + strconv.Itoa(p.Money)
}

type Table struct {
	deck    Deck
	dealer  Hand
	Players []Player
	minBet  int
	status  int
}

func NewTable() Table {
	deck := makeDeck(1)
	deck.shuffle()

	return Table{
		deck:    deck,
		dealer:  Hand{},
		Players: []Player{},
		minBet:  10,
	}
}

func (t *Table) EnterBet(player int, bet int) {
	if bet < t.minBet || bet > t.Players[player].Money {
		return
	}

	t.Players[player].Money -= bet
	t.Players[player].Hands = append(t.Players[player].Hands, Hand{Bet: bet, Active: true})
}

func (t *Table) ResetHands() {
	t.dealer = Hand{}
	for i := range t.Players {
		t.Players[i].Hands = []Hand{}
	}
}

func (t *Table) dealAll() {
	totalHands := 1
	for i := range t.Players {
		totalHands += len(t.Players[i].Hands)
	}

	for range 2 {
		for i := range t.Players {
			for j := range t.Players[i].Hands {
				t.deck.dealTo(&t.Players[i].Hands[j])
			}
		}
		t.deck.dealTo(&t.dealer)
	}
}

func (t *Table) PlayRound() {
	fmt.Printf("Dealer: %v + Hole\n", t.dealer.Cards[1])

	// check for dealer blackjack
	if t.dealer.hasBlackjack() {
		fmt.Println("Dealer has blackjack!")
		t.dealerTurn()
		return
	}

	reader := bufio.NewReader(os.Stdin)

	// player turn
	for i := range t.Players {
		player := &t.Players[i]
		for j := 0; j < len(player.Hands); j++ {
			current := &player.Hands[j]
			printHand(i, j, current)

			// fill in cards (i.e. previously split)
			for range 2 - len(current.Cards) {
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
		for i := range t.Players {
			player := &t.Players[i]
			for j := range t.Players[i].Hands {
				current := &player.Hands[j]

				if current.Active && !current.hasBust() {
					player.Money += 2 * current.Bet
				}
			}
		}
	} else {
		dealerScore := t.dealer.bestScore()
		for i := range t.Players {
			player := &t.Players[i]
			for j := range t.Players[i].Hands {
				current := &player.Hands[j]

				if !current.Active {
					continue
				}

				switch playerScore := current.bestScore(); {
				case playerScore > dealerScore:
					fmt.Printf("Player %v Hand %v wins!\n", i, j)
					player.Money += 2 * current.Bet
				case playerScore == dealerScore:
					fmt.Printf("Player %v Hand %v pushes!\n", i, j)
					player.Money += current.Bet
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
	return p.Money >= hand.Bet
}

// attempts to double a hand, returns whether double was successful
func double(hand *Hand, player *Player, deck *Deck) bool {
	if player.canDouble(hand) {
		deck.dealTo(hand)
		player.Money -= hand.Bet
		hand.Bet *= 2
		return true
	}
	return false
}

// attempts to split a hand, returns whether split was successful
func split(handIndex int, player *Player, deck *Deck) bool {
	oldHand := &player.Hands[handIndex]

	if oldHand.canSplit() && player.canDouble(oldHand) {
		newHand := Hand{Cards: []Card{oldHand.Cards[1]}, Bet: oldHand.Bet, Active: true}
		oldHand.Cards = oldHand.Cards[:1]
		deck.dealTo(oldHand)

		player.Hands = slices.Insert(player.Hands, handIndex+1, newHand)
		return true
	}
	return false
}

// checks if a hand has bust and takes money/sets hand inactive if it has
// returns whether bust was detected
func bust(hand *Hand) bool {
	if hand.hasBust() {
		fmt.Println("Player has bust!")
		hand.Bet = 0
		hand.Active = false
		return true
	}
	return false
}

// checks if a hand has blackjack and pays out/sets hand inactive if it does
// returns whether blackjack was detected
func blackjack(hand *Hand, player *Player) bool {
	if hand.hasBlackjack() {
		fmt.Println("Player has blackjack!")
		player.Money += (5 * hand.Bet) / 2
		hand.Bet = 0
		hand.Active = false
		return true
	}
	return false
}
