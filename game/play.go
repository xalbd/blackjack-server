package game

type Table struct {
	deck   Deck
	dealer Hand
	player Hand
}

func NewTable() Table {
	deck := Shuffle(NewDeck())

	return Table{
		deck:   deck,
		dealer: Hand{},
		player: Hand{},
	}
}

func Deal(d *Deck, h *Hand) {
	if dealt, ok := d.Deal(); ok {
		*h = append(*h, dealt)
	}
}

func (t *Table) Deal() {
	if len(t.deck) < 4 {
		t.deck = Shuffle(NewDeck())
	}

	for range 2 {
		Deal(&t.deck, &t.player)
		Deal(&t.deck, &t.dealer)
	}
}
