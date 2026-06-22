package orderbook

import (
	"reflect"
	"testing"
)

func TestOrderBook_TopBids(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	// Populate Bids (Buy orders)
	// Bids should be sorted in descending order (highest price first)
	orders := []Order{
		{Side: "buy", Price: 100.0, Amount: 1.5},
		{Side: "buy", Price: 100.0, Amount: 0.5}, // Same price level, should accumulate to 2.0
		{Side: "buy", Price: 102.5, Amount: 0.8}, // Highest bid
		{Side: "buy", Price: 99.0,  Amount: 3.0}, // Lowest bid
	}

	for _, order := range orders {
		ob.Add(order)
	}

	// 3. Define test cases for different depths
	tests := []struct {
		name  string
		depth int
		want  [][]float64
	}{
		{
			name:  "Request depth larger than available levels",
			depth: 5,
			want: [][]float64{
				{102.5, 0.8}, // Highest price
				{100.0, 2.0}, // Accumulated amount (1.5 + 0.5)
				{99.0,  3.0}, // Lowest price
			},
		},
		{
			name:  "Request depth smaller than available levels",
			depth: 2,
			want: [][]float64{
				{102.5, 0.8},
				{100.0, 2.0},
			},
		},
		{
			name:  "Request depth of 0",
			depth: 0,
			want:  [][]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ob.topBids(tt.depth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("topBids() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOrderBook_TopAsks(t *testing.T) {
	// 1. Initialize OrderBook
	ob := NewOrderBook("BTC", "USDT")

	// 2. Populate Asks (Sell orders)
	// Asks should be sorted in ascending order (lowest price first)
	orders := []Order{
		{Side: "sell", Price: 105.0, Amount: 2.5},
		{Side: "sell", Price: 105.0, Amount: 1.5}, // Same price level, should accumulate to 4.0
		{Side: "sell", Price: 103.0, Amount: 0.5}, // Lowest ask
		{Side: "sell", Price: 107.5, Amount: 1.0}, // Highest ask
	}

	for _, order := range orders {
		ob.Add(order)
	}

	// Define test cases for different depths
	tests := []struct {
		name  string
		depth int
		want  [][]float64
	}{
		{
			name:  "Request depth larger than available levels",
			depth: 5,
			want: [][]float64{
				{103.0, 0.5}, // Lowest price
				{105.0, 4.0}, // Accumulated amount (2.5 + 1.5)
				{107.5, 1.0}, // Highest price
			},
		},
		{
			name:  "Request depth smaller than available levels",
			depth: 2,
			want: [][]float64{
				{103.0, 0.5},
				{105.0, 4.0},
			},
		},
		{
			name:  "Request depth of 0",
			depth: 0,
			want:  [][]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ob.topAsks(tt.depth)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("topAsks() = %v, want %v", got, tt.want)
			}
		})
	}
}