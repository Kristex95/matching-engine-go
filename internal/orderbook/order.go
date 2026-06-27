package orderbook

import "github.com/shopspring/decimal"

type Order struct {
	ID     string
	Side   string
	Type   string
	Price  decimal.Decimal
	Amount decimal.Decimal
}

type OrderUpdate struct {
	OrderID         string          `json:"order_id"`
	Status          string          `json:"status"` // "filled" or "partially_filled" or "cancelled"
	FilledAmount    decimal.Decimal `json:"filled_amount"`
	RemainingAmount decimal.Decimal `json:"remaining_amount"`
}