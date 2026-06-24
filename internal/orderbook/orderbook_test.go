package orderbook

import (
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
)

func TestOrderBook_TopBids(t *testing.T) {
	ob := NewOrderBook("BTC", "USDT")

	orders := []Order{
		{Side: "buy", Price: decimal.NewFromFloat(100.0), Amount: decimal.NewFromFloat(1.5)},
		{Side: "buy", Price: decimal.NewFromFloat(100.0), Amount: decimal.NewFromFloat(0.5)},
		{Side: "buy", Price: decimal.NewFromFloat(102.5), Amount: decimal.NewFromFloat(0.8)},
		{Side: "buy", Price: decimal.NewFromFloat(99.0),  Amount: decimal.NewFromFloat(3.0)},
	}

	for _, order := range orders {
		ob.Add(order)
	}

	tests := []struct {
		name  string
		depth int
		want  [][]string
	}{
		{
			name:  "Request depth larger than available levels",
			depth: 5,
			want: [][]string{
				{"102.5", "0.8"},
				{"100", "2"},
				{"99", "3"},
			},
		},
		{
			name:  "Request depth smaller than available levels",
			depth: 2,
			want: [][]string{
				{"102.5", "0.8"},
				{"100", "2"},
			},
		},
		{
			name:  "Request depth of 0",
			depth: 0,
			want:  [][]string{},
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
	ob := NewOrderBook("BTC", "USDT")

	orders := []Order{
		{Side: "sell", Price: decimal.NewFromFloat(105.0), Amount: decimal.NewFromFloat(2.5)},
		{Side: "sell", Price: decimal.NewFromFloat(105.0), Amount: decimal.NewFromFloat(1.5)},
		{Side: "sell", Price: decimal.NewFromFloat(103.0), Amount: decimal.NewFromFloat(0.5)},
		{Side: "sell", Price: decimal.NewFromFloat(107.5), Amount: decimal.NewFromFloat(1.0)},
	}

	for _, order := range orders {
		ob.Add(order)
	}

	tests := []struct {
		name  string
		depth int
		want  [][]string
	}{
		{
			name:  "Request depth larger than available levels",
			depth: 5,
			want: [][]string{
				{"103", "0.5"},
				{"105", "4"},
				{"107.5", "1"},
			},
		},
		{
			name:  "Request depth smaller than available levels",
			depth: 2,
			want: [][]string{
				{"103", "0.5"},
				{"105", "4"},
			},
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