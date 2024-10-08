package game

import (
	"encoding/json"
	"slices"
	"time"
)

type tableStatus int

const (
	Betting tableStatus = iota
	PlayerTurn
	DealerTurn
)

type player struct {
	UID         string `json:"id"`
	DisplayName string `json:"displayName"`
	Money       int64  `json:"money"`
	active      bool
}

type table struct {
	deck                  Deck
	dealer                Hand
	minBet                int64
	seats                 int
	status                tableStatus
	Players               []player
	Hands                 []Hand
	ActiveHand            int
	actionTimeStart       time.Time
	beginBettingTimeLimit bool
	moveTimeLimit         time.Duration
	bettingTimeLimit      time.Duration
	Broadcast             chan []byte
	getMoney              func(string) int64
	deltaMoney            func(string, int64)
}

func newTable(broadcast chan []byte, seats int, getMoney func(string) int64, deltaMoney func(string, int64)) table {
	deck := makeDeck(1)
	deck.shuffle()

	t := table{
		deck:                  deck,
		dealer:                Hand{},
		minBet:                10,
		seats:                 seats,
		status:                Betting,
		Players:               []player{},
		Hands:                 make([]Hand, seats),
		actionTimeStart:       time.Now(),
		beginBettingTimeLimit: false,
		moveTimeLimit:         5 * time.Second,
		bettingTimeLimit:      15 * time.Second,
		ActiveHand:            -1,
		Broadcast:             broadcast,
		getMoney:              getMoney,
		deltaMoney:            deltaMoney,
	}

	return t
}

func (t *table) resetHands() {
	t.ActiveHand = -1
	t.status = Betting
	t.dealer = Hand{}
	t.beginBettingTimeLimit = false

	// reset hands and remove split hands
	h := 0
	for _, x := range t.Hands {
		if !x.Split {
			x.Cards = nil
			x.Bet = 0
			t.Hands[h] = x
			h++
		}
	}
	t.Hands = t.Hands[:h]

	// remove players who have left
	p := 0
	for _, x := range t.Players {
		if x.active {
			t.Players[p] = x
			p++
		}
	}
	t.Players = t.Players[:p]

	// clean up hands of players that left
	for i := range t.Hands {
		if t.playerWithUID(t.Hands[i].PlayerUID) == nil {
			t.Hands[i].PlayerUID = ""
		}
	}

	t.broadcast()
}

// call this method to broadcast table status to all players
func (t *table) broadcast() {
	var d []card

	// only show dealer's first card during player turn
	if t.status == PlayerTurn {
		d = t.dealer.Cards[:1]
	} else {
		d = t.dealer.Cards
	}

	for i := range t.Players {
		t.Players[i].Money = t.getMoney(t.Players[i].UID)
	}

	out, _ := json.Marshal(broadcast{
		Dealer:      d,
		Players:     t.Players,
		Hands:       t.Hands,
		ActiveHand:  t.ActiveHand,
		TableStatus: t.status,
		Time:        t.actionTimeStart.UnixMilli(),
	})
	t.Broadcast <- out
}

func (t *table) playerWithUID(uid string) *player {
	for i := range t.Players {
		if t.Players[i].UID == uid {
			return &t.Players[i]
		}
	}
	return nil
}

func (t *table) currentHand() *Hand {
	return &t.Hands[t.ActiveHand]
}

func (t *table) join(uid string, seat int) {
	if seat < 0 || seat >= t.seats || t.Hands[seat].PlayerUID != "" || t.getMoney(uid) < t.minBet {
		return
	}

	t.Hands[seat].PlayerUID = uid
	t.broadcast()
}

func (t *table) leave(uid string, seat int) {
	if seat < 0 || seat >= t.seats || t.Hands[seat].PlayerUID != uid || t.Hands[seat].Bet > 0 {
		return
	}

	t.Hands[seat] = Hand{}
	t.broadcast()
}

func (t *table) enterBet(uid string, bet int64, seat int) {
	player := t.playerWithUID(uid)

	if player == nil || bet < t.minBet || bet > t.getMoney(uid) || seat < 0 || seat >= t.seats || t.Hands[seat].PlayerUID != uid || t.Hands[seat].Bet > 0 {
		return
	}

	// set flag for main game loop to start betting time limit once someone has put an initial bet in
	if !t.someBetsIn() {
		t.beginBettingTimeLimit = true
		t.actionTimeStart = time.Now()
	}

	t.deltaMoney(uid, -bet)
	t.Hands[seat].Bet = bet

	t.broadcast()
}

func (t *table) dealAll() {
	for range 2 {
		for i := range t.Hands {
			if t.Hands[i].PlayerUID != "" {
				t.deck.dealTo(&t.Hands[i])
			}
		}
		t.deck.dealTo(&t.dealer)
	}
}

func (t *table) someBetsIn() bool {
	for i := range t.Hands {
		if t.Hands[i].Bet != 0 {
			return true
		}
	}

	return false
}

func (t *table) seatsTaken() int {
	seats := 0
	for i := range t.Hands {
		if t.Hands[i].PlayerUID != "" {
			seats++
		}
	}
	return seats
}

func (t *table) allBetsIn() bool {
	bets := 0

	for i := range t.Hands {
		// return false if someone claimed a seat and hasn't finished betting
		if t.Hands[i].PlayerUID != "" && t.Hands[i].Bet == 0 {
			return false
		}

		// keep track of how many players have finished betting
		if t.Hands[i].Bet != 0 {
			bets++
		}
	}

	return bets > 0
}

func (t *table) startPlayerTurn() {
	t.status = PlayerTurn
	t.dealAll()
	if t.dealer.hasBlackjack() {
		t.dealerTurn()
	} else {
		t.advanceHand()
	}
}

func (t *table) dealerTurn() {
	t.status = DealerTurn
	t.broadcast()
	for !t.dealer.hasBust() && t.dealer.bestScore() < 17 {
		t.deck.dealTo(&t.dealer)
		t.broadcast()
	}

	d := t.dealer.bestScore()
	for i, h := range t.Hands {
		if h.PlayerUID == "" {
			continue
		}

		p := t.playerWithUID(h.PlayerUID)

		if h.bestScore() > d {
			t.deltaMoney(p.UID, 2*h.Bet)
		} else if h.bestScore() == d {
			t.deltaMoney(p.UID, h.Bet)
		}

		t.Hands[i].Bet = 0
		t.broadcast()
	}

	t.resetHands()
}

// deals a card to the current hand
func (t *table) hit() bool {
	t.deck.dealTo(t.currentHand())
	return true
}

// returns whether current hand can be doubled
func (t *table) canDouble() bool {
	hand := t.currentHand()
	return t.getMoney(hand.PlayerUID) >= hand.Bet
}

// returns whether current hand can be split
func (t *table) canSplit() bool {
	hand := t.currentHand()
	return t.canDouble() && len(hand.Cards) == 2 && hand.Cards[0].value() == hand.Cards[1].value()
}

// attempts to double current hand, returns whether double was successful
func (t *table) double() bool {
	hand := t.currentHand()
	player := t.playerWithUID(hand.PlayerUID)

	if t.canDouble() {
		t.hit()
		t.deltaMoney(player.UID, -hand.Bet)
		hand.Bet *= 2
		return true
	}
	return false
}

// attempts to split current hand, returns whether split was successful
func (t *table) split() bool {
	oldHand := t.currentHand()

	if t.canSplit() {
		newHand := Hand{Cards: []card{oldHand.Cards[1]}, Bet: oldHand.Bet, PlayerUID: oldHand.PlayerUID, Split: true}
		oldHand.Cards = oldHand.Cards[:1]
		t.Hands = slices.Insert(t.Hands, t.ActiveHand+1, newHand)
		return true
	}
	return false
}

// advances active hand as far as possible
func (table *table) advanceHand() {
	table.ActiveHand++

	// skip past empty slots
	for table.ActiveHand < len(table.Hands) && (table.Hands[table.ActiveHand].PlayerUID == "" || table.Hands[table.ActiveHand].Bet == 0) {
		table.ActiveHand++
	}

	// new player's turn is active
	table.actionTimeStart = time.Now()
	table.broadcast()

	if table.ActiveHand >= len(table.Hands) {
		table.dealerTurn()
	} else {
		// check for bust/blackjack and then skip for inactive players
		if table.bust() || table.blackjack() || !table.playerWithUID(table.Hands[table.ActiveHand].PlayerUID).active {
			table.advanceHand()
		}
	}
}

// checks if current hand has bust and takes money if it has
// returns whether bust was detected
func (t *table) bust() bool {
	if t.currentHand().hasBust() {
		t.currentHand().Bet = 0
		return true
	}
	return false
}

// checks if current hand has blackjack and pays out if it does
// returns whether blackjack was detected
func (t *table) blackjack() bool {
	hand := t.currentHand()
	player := t.playerWithUID(hand.PlayerUID)

	if hand.hasBlackjack() {
		t.deltaMoney(player.UID, 5*hand.Bet/2)
		hand.Bet = 0
		return true
	}
	return false
}
