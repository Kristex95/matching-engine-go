package orderbook

import (
	"sort"
)

func (ob *OrderBook) Match(order *Order) []Trade {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []Trade
	onTradeWrapper := func(taker, maker string, price, qty float64) {
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

func (ob *OrderBook) matchMarket(order *Order, onTrade func(string, string, float64, float64)) {
	if order.Side == "buy" {
		ob.matchMarketBuy(order, onTrade)
	} else {
		ob.matchMarketSell(order, onTrade)
	}
}

func (ob *OrderBook) matchLimit(order *Order, onTrade func(string, string, float64, float64)) {
	if order.Side == "buy" {
		ob.matchLimitBuy(order, onTrade)
	} else {
		ob.matchLimitSell(order, onTrade)
	}
}

func (ob *OrderBook) match(
	order *Order,
	book map[float64]*PriceLevel,
	sortAsc bool,
	priceOK func(orderPrice, bookPrice float64) bool,
	onTrade func(takerID, makerID string, price, qty float64),
	onUnfilled func(*Order),
) {
	var prices []float64
	for p := range book {
		prices = append(prices, p)
	}

	sort.Slice(prices, func(i, j int) bool {
		if sortAsc {
			return prices[i] < prices[j]
		}
		return prices[i] > prices[j]
	})

	remaining := order.Amount

	for _, price := range prices {
		if remaining <= 0 {
			break
		}

		level := book[price]
		newQueue := []*Order{}

		for _, bookOrder := range level.Orders {

			if remaining <= 0 {
				newQueue = append(newQueue, bookOrder)
				continue
			}

			if priceOK(order.Price, bookOrder.Price) {
				matchQty := min(remaining, bookOrder.Amount)

				onTrade(order.ID, bookOrder.ID, bookOrder.Price, matchQty)

				bookOrder.Amount -= matchQty
				remaining -= matchQty

				if bookOrder.Amount > 0 {
					newQueue = append(newQueue, bookOrder)
				}
			} else {
				newQueue = append(newQueue, bookOrder)
			}
		}

		level.Orders = newQueue

		if len(level.Orders) == 0 {
			delete(book, price)
		}
	}

	if remaining > 0 {
		order.Amount = remaining
		onUnfilled(order)
	}
}

// limit orders
func (ob *OrderBook) matchLimitBuy(order *Order, onTrade func(string, string, float64, float64)) {
	ob.match(order, ob.Asks, true,
		func(orderPrice, askPrice float64) bool { return orderPrice >= askPrice },
		onTrade,
		func(o *Order) { ob.Add(*o) },
	)
}

func (ob *OrderBook) matchLimitSell(order *Order, onTrade func(string, string, float64, float64)) {
	ob.match(order, ob.Bids, false,
		func(orderPrice, bidPrice float64) bool { return orderPrice <= bidPrice },
		onTrade,
		func(o *Order) { ob.Add(*o) },
	)
}

// market orders
func (ob *OrderBook) matchMarketBuy(order *Order, onTrade func(string, string, float64, float64)) {
	ob.match(order, ob.Asks, true,
		func(_, _ float64) bool { return true },
		onTrade,
		func(o *Order) {},
	)
}

func (ob *OrderBook) matchMarketSell(order *Order, onTrade func(string, string, float64, float64)) {
	ob.match(order, ob.Bids, false,
		func(_, _ float64) bool { return true },
		onTrade,
		func(o *Order) {},
	)
}
