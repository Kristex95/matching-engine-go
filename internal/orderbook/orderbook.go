package orderbook

import (
	"sort"
	"sync"

	"github.com/shopspring/decimal"
)

type OrderBook struct {
	mu            sync.Mutex
	BaseCurrency  string
	QuoteCurrency string
	Bids          map[string]*PriceLevel
	Asks          map[string]*PriceLevel
}

func NewOrderBook(base, quote string) *OrderBook {
	return &OrderBook{
		BaseCurrency:  base,
		QuoteCurrency: quote,
		Bids:          make(map[string]*PriceLevel),
		Asks:          make(map[string]*PriceLevel),
	}
}

type PriceLevel struct {
	Price  decimal.Decimal
	Orders []*Order
}

func (ob *OrderBook) Add(order Order) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.add(order)
}

func (ob *OrderBook) add(order Order) {
	level := ob.getLevel(order)
	
	heapOrder := order
	level.Orders = append(level.Orders, &heapOrder)
}

func (ob *OrderBook) getLevel(order Order) *PriceLevel {
	var book map[string]*PriceLevel

	if order.Side == "buy" {
		book = ob.Bids
	} else {
		book = ob.Asks
	}

	priceKey := order.Price.String()

	level, ok := book[priceKey]
	if !ok {
		level = &PriceLevel{
			Price:  order.Price,
			Orders: []*Order{},
		}
		book[priceKey] = level
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

func (ob *OrderBook) topBids(depth int) [][]string {
	prices := make([]decimal.Decimal, 0, len(ob.Bids))

	for priceStr := range ob.Bids {
		d, _ := decimal.NewFromString(priceStr)
		prices = append(prices, d)
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].GreaterThan(prices[j])
	})

	if len(prices) > depth {
		prices = prices[:depth]
	}

	levels := make([][]string, 0, len(prices))
	for _, price := range prices {
		level := ob.Bids[price.String()]
		levels = append(levels, []string{
			price.String(),
			level.TotalAmount().String(),
		})
	}
	return levels
}

func (ob *OrderBook) topAsks(depth int) [][]string {
	prices := make([]decimal.Decimal, 0, len(ob.Asks))

	for priceStr := range ob.Asks {
		d, _ := decimal.NewFromString(priceStr)
		prices = append(prices, d)
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].LessThan(prices[j])
	})

	if len(prices) > depth {
		prices = prices[:depth]
	}

	levels := make([][]string, 0, len(prices))
	for _, price := range prices {
		level := ob.Asks[price.String()]
		levels = append(levels, []string{
			price.String(),
			level.TotalAmount().String(),
		})
	}
	return levels
}

func (pl *PriceLevel) TotalAmount() decimal.Decimal {
	total := decimal.Zero
	for _, order := range pl.Orders {
		total = total.Add(order.Amount)
	}
	return total
}