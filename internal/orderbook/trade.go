package orderbook

import "github.com/shopspring/decimal"

type Trade struct {
	TakerOrderID  string  			`json:"taker_order_id"`
	MakerOrderID  string  			`json:"maker_order_id"`
	Price         decimal.Decimal 	`json:"price"`
	Amount        decimal.Decimal 	`json:"amount"`
	Side          string  			`json:"side"` // "buy" or "sell" relative to the taker
	BaseCurrency  string  			`json:"base_currency"`
	QuoteCurrency string  			`json:"quote_currency"`
}
