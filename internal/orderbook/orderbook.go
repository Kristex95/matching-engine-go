package orderbook

import "sync"

type OrderBook struct {
	mu sync.Mutex
	Bids map[float64]*PriceLevel
	Asks map[float64]*PriceLevel
}

func NewOrderBook() *OrderBook {
	return &OrderBook{
		Bids: make(map[float64]*PriceLevel),
		Asks: make(map[float64]*PriceLevel),
	}
}

type PriceLevel struct {
	Price  float64
	Orders []*Order
}

func (ob *OrderBook) Add(order Order) {
	level := ob.getLevel(order)
	level.Orders = append(level.Orders, &order)
}

func (ob *OrderBook) getLevel(order Order) *PriceLevel {

	var book map[float64]*PriceLevel

	if order.Side == "buy" {
		book = ob.Bids
	} else {
		book = ob.Asks
	}

	level, ok := book[order.Price]
	if !ok {
		level = &PriceLevel{
			Price:  order.Price,
			Orders: []*Order{},
		}
		book[order.Price] = level
	}

	return level
}