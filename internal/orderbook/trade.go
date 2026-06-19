package orderbook

type Trade struct {
	TakerOrderID  string  `json:"taker_order_id"`
	MakerOrderID  string  `json:"maker_order_id"`
	Price         float64 `json:"price"`
	Amount        float64 `json:"amount"`
	Side          string  `json:"side"` // "buy" or "sell" relative to the taker
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
}
