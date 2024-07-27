package game

import "strconv"

func (table *Table) HandleCommand(p int, message string) {
	switch table.status {
	case 0: // betting
		switch message[0] {
		case 'b': // add a bet
			bet, _ := strconv.Atoi(message[1:])
			table.EnterBet(p, bet)
		case 'e': // end betting, deal, and check for dealer blackjack (temporary for testing)
			table.dealAll()
			table.status++
		}
	case 1: // player turn
		switch message[0] {
		case 'h': // hit
		case 's': // stand
		case 'd': // double
		case 'p': // split
		case 'e': // end player turn (temporary for testing) and dealer turn
			table.status++
		}
	}
}
