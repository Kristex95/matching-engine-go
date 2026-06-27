package orderbook

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestLimitOrderMatching(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// 1. Add limit sells (liquidity)
	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: decimal.NewFromInt(50000), Amount: decimal.NewFromFloat(1.0)})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: decimal.NewFromInt(51000), Amount: decimal.NewFromFloat(2.0)})

	// 2. Taker Buy order: 1.5 @ 51000
	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: decimal.NewFromInt(51000), Amount: decimal.NewFromFloat(1.5)}
	trades, _ := ob.Match(buy)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	if trades[0].MakerOrderID != "s1" || !trades[0].Amount.Equal(decimal.NewFromFloat(1.0)) || !trades[0].Price.Equal(decimal.NewFromInt(50000)) {
		t.Errorf("first trade incorrect: %+v", trades[0])
	}

	if trades[1].MakerOrderID != "s2" || !trades[1].Amount.Equal(decimal.NewFromFloat(0.5)) || !trades[1].Price.Equal(decimal.NewFromInt(51000)) {
		t.Errorf("second trade incorrect: %+v", trades[1])
	}

	// 3. Verify Order Book state (Changed map keys to string)
	if _, ok := ob.Asks[decimal.NewFromInt(50000).String()]; ok {
		t.Error("price level 50000 should be removed after being cleared")
	}
	if level, ok := ob.Asks[decimal.NewFromInt(51000).String()]; ok {
		if len(level.Orders) != 1 || !level.Orders[0].Amount.Equal(decimal.NewFromFloat(1.5)) {
			t.Errorf("expected 1.5 remaining in s2, got %s", level.Orders[0].Amount)
		}
	} else {
		t.Error("price level 51000 should still exist")
	}
}

func TestMarketOrderMatching(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	ob.Match(&Order{ID: "b1", Side: "buy", Type: "limit", Price: decimal.NewFromInt(49000), Amount: decimal.NewFromFloat(1.0)})
	ob.Match(&Order{ID: "b2", Side: "buy", Type: "limit", Price: decimal.NewFromInt(48000), Amount: decimal.NewFromFloat(1.0)})

	sell := &Order{ID: "s1", Side: "sell", Type: "market", Amount: decimal.NewFromFloat(1.5)}
	trades, _ := ob.Match(sell)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	if !trades[0].Price.Equal(decimal.NewFromInt(49000)) || !trades[0].Amount.Equal(decimal.NewFromFloat(1.0)) {
		t.Errorf("first trade mismatch: %+v", trades[0])
	}
	if !trades[1].Price.Equal(decimal.NewFromInt(48000)) || !trades[1].Amount.Equal(decimal.NewFromFloat(0.5)) {
		t.Errorf("second trade mismatch: %+v", trades[1])
	}
}

func TestFIFOWithinPriceLevel(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.0)})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.0)})

	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.5)}
	trades, _ := ob.Match(buy)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	if trades[0].MakerOrderID != "s1" {
		t.Errorf("expected s1 to be filled first (FIFO), got %s", trades[0].MakerOrderID)
	}
}

func TestSequentialBuyOrdersMatchRestingSells(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.0)})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.0)})

	firstBuy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(0.75)}
	firstTrades, _ := ob.Match(firstBuy)

	if len(firstTrades) != 1 {
		t.Fatalf("expected 1 trade from first buy, got %d", len(firstTrades))
	}

	secondBuy := &Order{ID: "b2", Side: "buy", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(0.75)}
	secondTrades, _ := ob.Match(secondBuy)

	if len(secondTrades) != 2 {
		t.Fatalf("expected 2 trades from second buy, got %d", len(secondTrades))
	}
}

func TestLimitOrderPartiallyMatchedAndAddedToBook(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.0)})

	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: decimal.NewFromInt(100), Amount: decimal.NewFromFloat(1.5)}
	trades, _ := ob.Match(buy)

	if len(trades) != 1 || !trades[0].Amount.Equal(decimal.NewFromFloat(1.0)) {
		t.Fatalf("expected 1.0 match, got %+v", trades)
	}

	// Changed map key lookup to string format
	if level, ok := ob.Bids[decimal.NewFromInt(100).String()]; !ok || len(level.Orders) == 0 || !level.Orders[0].Amount.Equal(decimal.NewFromFloat(0.5)) {
		t.Error("remaining 0.5 of buy order should be resting in Bids")
	}
}

func TestPriceImprovementMatching(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: decimal.NewFromInt(10000), Amount: decimal.NewFromFloat(1.0)})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: decimal.NewFromInt(11000), Amount: decimal.NewFromFloat(1.0)})

	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: decimal.NewFromInt(11000), Amount: decimal.NewFromFloat(2.0)}
	trades, _ := ob.Match(buy)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	if trades[0].MakerOrderID != "s1" || !trades[0].Amount.Equal(decimal.NewFromFloat(1.0)) || !trades[0].Price.Equal(decimal.NewFromInt(10000)) {
		t.Errorf("first trade should match s1 at 10000: %+v", trades[0])
	}
}