package orderbook

import (
	"sort"

	"github.com/shopspring/decimal"
)

func (ob *OrderBook) Match(order *Order) []Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []Trade
	onTradeWrapper := func(taker, maker string, price, qty decimal.Decimal) {
		trades = append(trades, Trade{
			TakerOrderID:  taker,
			MakerOrderID:  maker,
			Price:         price,
			Amount:        qty,
			Side:          order.Side,
			BaseCurrency:  ob.BaseCurrency,
			QuoteCurrency: ob.QuoteCurrency,
		})
	}

	switch order.Type {
	case "market":
		ob.matchMarket(order, onTradeWrapper)
	case "limit":
		ob.matchLimit(order, onTradeWrapper)
	}

	return trades
}

func (ob *OrderBook) matchMarket(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal)) {
	if order.Side == "buy" {
		ob.matchMarketBuy(order, onTrade)
	} else {
		ob.matchMarketSell(order, onTrade)
	}
}

func (ob *OrderBook) matchLimit(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal)) {
	if order.Side == "buy" {
		ob.matchLimitBuy(order, onTrade)
	} else {
		ob.matchLimitSell(order, onTrade)
	}
}

func (ob *OrderBook) match(
	order *Order,
	book map[string]*PriceLevel,
	sortAsc bool,
	priceOK func(orderPrice, bookPrice decimal.Decimal) bool,
	onTrade func(takerID, makerID string, price, qty decimal.Decimal),
	onUnfilled func(*Order),
) {
	prices := make([]decimal.Decimal, 0, len(book))
	for pStr := range book {
		p, err := decimal.NewFromString(pStr)
		if err == nil {
			prices = append(prices, p)
		}
	}

	sort.Slice(prices, func(i, j int) bool {
		if sortAsc {
			return prices[i].LessThan(prices[j])
		}
		return prices[i].GreaterThan(prices[j])
	})

	remaining := order.Amount

	for _, price := range prices {
		if remaining.IsZero() || remaining.IsNegative() {
			break
		}

		priceKey := price.String()
		level := book[priceKey]
		newQueue := []*Order{}

		for _, bookOrder := range level.Orders {
			if remaining.IsZero() || remaining.IsNegative() {
				newQueue = append(newQueue, bookOrder)
				continue
			}

			if priceOK(order.Price, bookOrder.Price) {
				matchQty := decimal.Min(remaining, bookOrder.Amount)

				onTrade(order.ID, bookOrder.ID, bookOrder.Price, matchQty)

				bookOrder.Amount = bookOrder.Amount.Sub(matchQty)
				remaining = remaining.Sub(matchQty)

				if bookOrder.Amount.IsPositive() {
					newQueue = append(newQueue, bookOrder)
				}
			} else {
				newQueue = append(newQueue, bookOrder)
			}
		}

		level.Orders = newQueue

		if len(level.Orders) == 0 {
			delete(book, priceKey)
		}
	}

	if remaining.IsPositive() {
		order.Amount = remaining
		onUnfilled(order)
	}
}

// limit orders
func (ob *OrderBook) matchLimitBuy(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal)) {
	ob.match(order, ob.Asks, true,
		func(orderPrice, askPrice decimal.Decimal) bool { return orderPrice.GreaterThanOrEqual(askPrice) },
		onTrade,
		func(o *Order) { ob.add(*o) },
	)
}

func (ob *OrderBook) matchLimitSell(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal)) {
	ob.match(order, ob.Bids, false,
		func(orderPrice, bidPrice decimal.Decimal) bool { return orderPrice.LessThanOrEqual(bidPrice) },
		onTrade,
		func(o *Order) { ob.add(*o) },
	)
}

// market orders
func (ob *OrderBook) matchMarketBuy(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal)) {
	ob.match(order, ob.Asks, true,
		func(_, _ decimal.Decimal) bool { return true },
		onTrade,
		func(o *Order) {},
	)
}

func (ob *OrderBook) matchMarketSell(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal)) {
	ob.match(order, ob.Bids, false,
		func(_, _ decimal.Decimal) bool { return true },
		onTrade,
		func(o *Order) {},
	)
}