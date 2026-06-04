package orderbook

type Order struct {
	ID     string
	Side   string
	Type   string
	Price  float64
	Amount float64
}