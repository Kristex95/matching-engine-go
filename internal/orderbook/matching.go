package orderbook

import (
	"fmt"
	"sort"
)

func (ob *OrderBook) Match(order *Order) {
	ob.mu.Lock()

	switch order.Type {
	case "market":
		ob.matchMarket(order)
	case "limit":
		ob.matchLimit(order)
	}

	defer ob.mu.Unlock()
}

func (ob *OrderBook) matchMarket(order *Order) {
	if order.Side == "buy" {
		ob.matchMarketBuy(order)
	} else {
		ob.matchMarketSell(order)
	}
}

func (ob *OrderBook) matchLimit(order *Order) {
	if order.Side == "buy" {
		ob.matchLimitBuy(order)
	} else {
		ob.matchSell(order)
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
func (ob *OrderBook) matchLimitBuy(order *Order) {
	fmt.Println("=== MATCH BUY ===")

	ob.match(
		order,
		ob.Asks,
		true,
		func(orderPrice, askPrice float64) bool {
			return orderPrice >= askPrice
		},
		func(taker string, maker string, price float64, qty float64) {
			fmt.Printf(
				"TRADE BUY=%s SELL=%s PRICE=%f QTY=%f\n",
				taker, maker, price, qty,
			)
		},
		func(o *Order) {
			ob.Add(*o)
		},
	)
}

func (ob *OrderBook) matchSell(order *Order) {
	fmt.Println("=== MATCH SELL ===")

	ob.match(
		order,
		ob.Bids,
		false,
		func(orderPrice, bidPrice float64) bool {
			return orderPrice <= bidPrice
		},
		func(taker string, maker string, price float64, qty float64) {
			fmt.Printf(
				"TRADE SELL=%s BUY=%s PRICE=%f QTY=%f\n",
				taker, maker, price, qty,
			)
		},
		func(o *Order) {
			ob.Add(*o)
		},
	)
}

// market orders
func (ob *OrderBook) matchMarketBuy(order *Order) {
	fmt.Println("=== MARKET BUY ===")

	ob.match(
		order,
		ob.Asks,
		true,
		func(_, _ float64) bool { return true },
		func(taker string, maker string, price float64, qty float64) {
			fmt.Printf(
				"TRADE MARKET BUY=%s SELL=%s PRICE=%f QTY=%f\n",
				taker, maker, price, qty,
			)
		},
		func(o *Order) {
			fmt.Println("MARKET ORDER PARTIALLY FILLED OR REJECTED")
		},
	)
}

func (ob *OrderBook) matchMarketSell(order *Order) {
	fmt.Println("=== MARKET SELL ===")

	ob.match(
		order,
		ob.Bids,
		false,
		func(_, _ float64) bool { return true },
		func(taker string, maker string, price float64, qty float64) {
			fmt.Printf(
				"TRADE MARKET SELL=%s BUY=%s PRICE=%f QTY=%f\n",
				taker, maker, price, qty,
			)
		},
		func(o *Order) {
			fmt.Println("MARKET ORDER PARTIALLY FILLED OR REJECTED")
		},
	)
}