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

	originalTakerAmount := order.Amount

	switch order.Type {
	case "market":
		ob.matchMarket(order, onTradeWrapper)
	case "limit":
		ob.matchLimit(order, onTradeWrapper)
	}

	takerFilled := originalTakerAmount.Sub(order.Amount)
	if takerFilled.IsPositive() {
		var takerStatus string
		if order.Amount.IsZero() {
			takerStatus = "filled"
		} else {
			takerStatus = "partially_filled"
		}
		
		orderUpdates = append(orderUpdates, OrderUpdate{
			OrderID:         order.ID,
			Status:          takerStatus,
			FilledAmount:    takerFilled,
			RemainingAmount: order.Amount,
		})
	} else if order.Type == "market" {
		orderUpdates = append(orderUpdates, OrderUpdate{
			OrderID:         order.ID,
			Status:          "cancelled", 
			FilledAmount:    decimal.Zero,
			RemainingAmount: order.Amount,
		})
	}

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
			scanBook(ob.Asks) // Taker bought, so maker was selling
		} else {
			scanBook(ob.Bids) // Taker sold, so maker was buying
		}

		status := "filled"
		if isStillInBook && remainingQty.IsPositive() {
			status = "partially_filled"
		}

		orderUpdates = append(orderUpdates, OrderUpdate{
			OrderID:         makerID,
			Status:          status,
			FilledAmount:    filledQty,
			RemainingAmount: remainingQty,
		})
	}

	return trades, orderUpdates
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

	order.Amount = remaining
	if remaining.IsPositive() {
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