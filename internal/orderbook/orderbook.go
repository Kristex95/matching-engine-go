package orderbook

import (
	"sort"
	"sync"
)

type OrderBook struct {
	mu            sync.Mutex
	BaseCurrency  string
	QuoteCurrency string
	Bids          map[float64]*PriceLevel
	Asks          map[float64]*PriceLevel
}

func NewOrderBook(base, quote string) *OrderBook {
	return &OrderBook{
		BaseCurrency:  base,
		QuoteCurrency: quote,
		Bids:          make(map[float64]*PriceLevel),
		Asks:          make(map[float64]*PriceLevel),
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

func (ob *OrderBook) Snapshot(depth int) Snapshot {
    ob.mu.Lock()
    defer ob.mu.Unlock()

    bids := ob.topBids(depth)
    asks := ob.topAsks(depth)

    return Snapshot{
        Symbol: ob.BaseCurrency + ob.QuoteCurrency,
        Bids:   bids,
        Asks:   asks,
    }
}

func (ob *OrderBook) topBids(depth int) [][]float64 {
    prices := make([]float64, 0, len(ob.Bids))

    for price := range ob.Bids {
        prices = append(prices, price)
    }

    sort.Sort(sort.Reverse(sort.Float64Slice(prices)))

    if len(prices) > depth {
        prices = prices[:depth]
    }

    levels := make([][]float64, 0, len(prices))

    for _, price := range prices {
        level := ob.Bids[price]

        levels = append(levels, []float64{
            price,
            level.TotalAmount(),
        })
    }

    return levels
}

func (ob *OrderBook) topAsks(depth int) [][]float64 {
	prices := make([]float64, 0, len(ob.Asks))

	for price := range ob.Asks {
		prices = append(prices, price)
	}

	sort.Float64s(prices)

	if len(prices) > depth {
		prices = prices[:depth]
	}

	levels := make([][]float64, 0, len(prices))

	for _, price := range prices {
		level := ob.Asks[price]

		levels = append(levels, []float64{
			price,
			level.TotalAmount(),
		})
	}

	return levels
}

func (pl *PriceLevel) TotalAmount() float64 {
	var total float64
	for _, order := range pl.Orders {
		total += order.Amount
	}
	return total
}
