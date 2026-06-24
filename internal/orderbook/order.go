package orderbook

import "github.com/shopspring/decimal"

type Order struct {
	ID     string
	Side   string
	Type   string
	Price  decimal.Decimal
	Amount decimal.Decimal
}