package orderbook

import (
	"testing"
)

func TestLimitOrderMatching(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// 1. Add limit sells (liquidity)
	// s1: 1.0 @ 50000
	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: 50000, Amount: 1.0})
	// s2: 2.0 @ 51000
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: 51000, Amount: 2.0})

	// 2. Taker Buy order: 1.5 @ 51000
	// This should match all of s1 (better price) and 0.5 of s2.
	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: 51000, Amount: 1.5}
	trades := ob.Match(buy)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	// Trade with s1 should happen first due to better price
	if trades[0].MakerOrderID != "s1" || trades[0].Amount != 1.0 || trades[0].Price != 50000 {
		t.Errorf("first trade incorrect: %+v", trades[0])
	}

	// Trade with s2 for the remainder
	if trades[1].MakerOrderID != "s2" || trades[1].Amount != 0.5 || trades[1].Price != 51000 {
		t.Errorf("second trade incorrect: %+v", trades[1])
	}

	// 3. Verify Order Book state
	if _, ok := ob.Asks[50000]; ok {
		t.Error("price level 50000 should be removed after being cleared")
	}
	if level, ok := ob.Asks[51000]; ok {
		if len(level.Orders) != 1 || level.Orders[0].Amount != 1.5 {
			t.Errorf("expected 1.5 remaining in s2, got %f", level.Orders[0].Amount)
		}
	} else {
		t.Error("price level 51000 should still exist")
	}
}

func TestMarketOrderMatching(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// Bids: 1.0 @ 49000 (b1), 1.0 @ 48000 (b2)
	ob.Match(&Order{ID: "b1", Side: "buy", Type: "limit", Price: 49000, Amount: 1.0})
	ob.Match(&Order{ID: "b2", Side: "buy", Type: "limit", Price: 48000, Amount: 1.0})

	// Market Sell 1.5. Should take all of b1 and 0.5 of b2.
	sell := &Order{ID: "s1", Side: "sell", Type: "market", Amount: 1.5}
	trades := ob.Match(sell)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	if trades[0].Price != 49000 || trades[0].Amount != 1.0 {
		t.Errorf("first trade mismatch: %+v", trades[0])
	}
	if trades[1].Price != 48000 || trades[1].Amount != 0.5 {
		t.Errorf("second trade mismatch: %+v", trades[1])
	}

	// Taker side in Trade struct should be "sell"
	if trades[0].Side != "sell" {
		t.Errorf("trade side should be 'sell' (relative to taker), got %s", trades[0].Side)
	}
}

func TestFIFOWithinPriceLevel(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// Two asks at same price
	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: 100, Amount: 1.0})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: 100, Amount: 1.0})

	// Buy 1.5 @ 100. Should fill s1 completely before s2 due to arrival order.
	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: 100, Amount: 1.5}
	trades := ob.Match(buy)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	if trades[0].MakerOrderID != "s1" {
		t.Errorf("expected s1 to be filled first (FIFO), got %s", trades[0].MakerOrderID)
	}
	if trades[1].MakerOrderID != "s2" || trades[1].Amount != 0.5 {
		t.Errorf("expected partial fill of s2, got %+v", trades[1])
	}
}

func TestSequentialBuyOrdersMatchRestingSells(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// Two sell orders at the same price level.
	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: 100, Amount: 1.0})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: 100, Amount: 1.0})

	// First buy partially fills s1.
	firstBuy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: 100, Amount: 0.75}
	firstTrades := ob.Match(firstBuy)

	if len(firstTrades) != 1 {
		t.Fatalf("expected 1 trade from first buy, got %d", len(firstTrades))
	}
	if firstTrades[0].MakerOrderID != "s1" || firstTrades[0].Amount != 0.75 || firstTrades[0].Price != 100 {
		t.Errorf("first buy should partially match s1: %+v", firstTrades[0])
	}

	// Second buy finishes s1, then partially fills s2.
	secondBuy := &Order{ID: "b2", Side: "buy", Type: "limit", Price: 100, Amount: 0.75}
	secondTrades := ob.Match(secondBuy)

	if len(secondTrades) != 2 {
		t.Fatalf("expected 2 trades from second buy, got %d", len(secondTrades))
	}
	if secondTrades[0].MakerOrderID != "s1" || secondTrades[0].Amount != 0.25 || secondTrades[0].Price != 100 {
		t.Errorf("second buy should finish s1 first: %+v", secondTrades[0])
	}
	if secondTrades[1].MakerOrderID != "s2" || secondTrades[1].Amount != 0.5 || secondTrades[1].Price != 100 {
		t.Errorf("second buy should partially match s2: %+v", secondTrades[1])
	}

	if level, ok := ob.Asks[100]; ok {
		if len(level.Orders) != 1 || level.Orders[0].ID != "s2" || level.Orders[0].Amount != 0.5 {
			t.Errorf("expected 0.5 remaining in s2, got %+v", level.Orders)
		}
	} else {
		t.Error("price level 100 should still exist")
	}
}

func TestLimitOrderPartiallyMatchedAndAddedToBook(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// Sell 1.0 @ 100
	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: 100, Amount: 1.0})

	// Buy 1.5 @ 100. Should match 1.0 and put the remaining 0.5 in Bids.
	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: 100, Amount: 1.5}
	trades := ob.Match(buy)

	if len(trades) != 1 || trades[0].Amount != 1.0 {
		t.Fatalf("expected 1.0 match, got %+v", trades)
	}

	if level, ok := ob.Bids[100]; !ok || len(level.Orders) == 0 || level.Orders[0].Amount != 0.5 {
		t.Error("remaining 0.5 of buy order should be resting in Bids")
	}
}

func TestPriceImprovementMatching(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// 1. Add the 2 limit sell orders (liquidity)
	ob.Match(&Order{ID: "s1", Side: "sell", Type: "limit", Price: 10000, Amount: 1.0})
	ob.Match(&Order{ID: "s2", Side: "sell", Type: "limit", Price: 11000, Amount: 1.0})

	// 2. Taker Buy order: 2.0 @ 11000
	buy := &Order{ID: "b1", Side: "buy", Type: "limit", Price: 11000, Amount: 2.0}
	trades := ob.Match(buy)

	if len(trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(trades))
	}

	// First trade must get the better price (10,000)
	if trades[0].MakerOrderID != "s1" || trades[0].Amount != 1.0 || trades[0].Price != 10000 {
		t.Errorf("first trade should match s1 at 10000: %+v", trades[0])
	}

	// Second trade matches the remaining amount at 11,000
	if trades[1].MakerOrderID != "s2" || trades[1].Amount != 1.0 || trades[1].Price != 11000 {
		t.Errorf("second trade should match s2 at 11000: %+v", trades[1])
	}

	// 3. Verify both price levels are completely cleared from the book
	if _, ok := ob.Asks[10000]; ok {
		t.Error("price level 10000 should be removed")
	}
	if _, ok := ob.Asks[11000]; ok {
		t.Error("price level 11000 should be removed")
	}
}
