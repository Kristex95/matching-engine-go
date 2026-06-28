package orderbook

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

func (ob *OrderBook) Match(order *Order) ([]Trade, []OrderUpdate) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var trades []Trade
	var orderUpdates []OrderUpdate
	
	makerFilledAmounts := make(map[string]decimal.Decimal)
	stpTriggered := false

	onTradeWrapper := func(taker, maker string, price, qty decimal.Decimal) {
		trades = append(trades, Trade{
			TakerOrderID:  taker,
			MakerOrderID:  maker,
			Price:         price,
			Amount:        qty,
			Side:          order.Side,
			BaseCurrency:  ob.BaseCurrency,
			QuoteCurrency: ob.QuoteCurrency,
			Timestamp:     time.Now().UnixMilli(),
		})
		
		makerFilledAmounts[maker] = makerFilledAmounts[maker].Add(qty)
	}

	// Handles STP Cancel Taker Event
	onSTPWrapper := func(taker *Order, maker *Order) bool {
		stpTriggered = true
		orderUpdates = append(orderUpdates, OrderUpdate{
			OrderID:         taker.ID,
			AccountID:       taker.AccountID,
			Status:          "cancelled", // Order is terminated due to STP
			FilledAmount:    decimal.Zero, 
			RemainingAmount: taker.Amount,
		})
		return true // Break out of matching
	}

	originalTakerAmount := order.Amount

	switch order.Type {
	case "market":
		ob.matchMarket(order, onTradeWrapper, onSTPWrapper)
	case "limit":
		ob.matchLimit(order, onTradeWrapper, onSTPWrapper)
	}

	// Taker order update logic
	takerFilled := originalTakerAmount.Sub(order.Amount)
	if !stpTriggered {
		if takerFilled.IsPositive() {
			var takerStatus string
			if order.Amount.IsZero() {
				takerStatus = "filled"
			} else {
				takerStatus = "partially_filled"
			}
			
			orderUpdates = append(orderUpdates, OrderUpdate{
				OrderID:         order.ID,
				AccountID:       order.AccountID,
				Status:          takerStatus,
				FilledAmount:    takerFilled,
				RemainingAmount: order.Amount,
			})
		} else if order.Type == "market" {
			orderUpdates = append(orderUpdates, OrderUpdate{
				OrderID:         order.ID,
				AccountID:       order.AccountID,
				Status:          "cancelled", 
				FilledAmount:    decimal.Zero,
				RemainingAmount: order.Amount,
			})
		}
	}

	// Process maker updates...
	for makerID, filledQty := range makerFilledAmounts {
		remainingQty := decimal.Zero
		isStillInBook := false

		scanBook := func(book map[string]*PriceLevel) {
			for _, level := range book {
				for _, bo := range level.Orders {
					if bo.ID == makerID {
						remainingQty = bo.Amount
						isStillInBook = true
						break
					}
				}
				if isStillInBook { break }
			}
		}

		if order.Side == "buy" {
			scanBook(ob.Asks)
		} else {
			scanBook(ob.Bids)
		}

		status := "filled"
		if isStillInBook && remainingQty.IsPositive() {
			status = "partially_filled"
		}

		orderUpdates = append(orderUpdates, OrderUpdate{
			OrderID:         makerID,
			AccountID:       order.AccountID,
			Status:          status,
			FilledAmount:    filledQty,
			RemainingAmount: remainingQty,
		})
	}

	return trades, orderUpdates
}

func (ob *OrderBook) matchMarket(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal), onSTP func(*Order, *Order) bool) {
	if order.Side == "buy" {
		ob.matchMarketBuy(order, onTrade, onSTP)
	} else {
		ob.matchMarketSell(order, onTrade, onSTP)
	}
}

func (ob *OrderBook) matchLimit(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal), onSTP func(*Order, *Order) bool) {
	if order.Side == "buy" {
		ob.matchLimitBuy(order, onTrade, onSTP)
	} else {
		ob.matchLimitSell(order, onTrade, onSTP)
	}
}

func (ob *OrderBook) match(
	order *Order,
	book map[string]*PriceLevel,
	sortAsc bool,
	priceOK func(orderPrice, bookPrice decimal.Decimal) bool,
	onTrade func(takerID, makerID string, price, qty decimal.Decimal),
	onSTP func(taker *Order, maker *Order) bool, // New callback
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
	stpTriggered := false

	for _, price := range prices {
		if remaining.IsZero() || remaining.IsNegative() || stpTriggered {
			break
		}

		priceKey := price.String()
		level := book[priceKey]
		newQueue := []*Order{}

		for _, bookOrder := range level.Orders {
			if remaining.IsZero() || remaining.IsNegative() || stpTriggered {
				newQueue = append(newQueue, bookOrder)
				continue
			}

			if priceOK(order.Price, bookOrder.Price) {
				// Self-Trade Prevention Check
				if order.AccountID == bookOrder.AccountID {
					order.Amount = remaining // Synchronize current remainder back to order struct
					if onSTP(order, bookOrder) {
						stpTriggered = true
						newQueue = append(newQueue, bookOrder)
						continue
					}
				}

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

	order.Amount = remaining
	// Only add to book or trigger unfilled actions if STP did not abort the order
	if remaining.IsPositive() && !stpTriggered {
		onUnfilled(order)
	}
}

// Limit Sub-methods
func (ob *OrderBook) matchLimitBuy(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal), onSTP func(*Order, *Order) bool) {
	ob.match(order, ob.Asks, true,
		func(orderPrice, askPrice decimal.Decimal) bool { return orderPrice.GreaterThanOrEqual(askPrice) },
		onTrade, onSTP,
		func(o *Order) { ob.add(*o) },
	)
}

func (ob *OrderBook) matchLimitSell(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal), onSTP func(*Order, *Order) bool) {
	ob.match(order, ob.Bids, false,
		func(orderPrice, bidPrice decimal.Decimal) bool { return orderPrice.LessThanOrEqual(bidPrice) },
		onTrade, onSTP,
		func(o *Order) { ob.add(*o) },
	)
}

// Market Sub-methods
func (ob *OrderBook) matchMarketBuy(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal), onSTP func(*Order, *Order) bool) {
	ob.match(order, ob.Asks, true,
		func(_, _ decimal.Decimal) bool { return true },
		onTrade, onSTP,
		func(o *Order) {},
	)
}

func (ob *OrderBook) matchMarketSell(order *Order, onTrade func(string, string, decimal.Decimal, decimal.Decimal), onSTP func(*Order, *Order) bool) {
	ob.match(order, ob.Bids, false,
		func(_, _ decimal.Decimal) bool { return true },
		onTrade, onSTP,
		func(o *Order) {},
	)
}