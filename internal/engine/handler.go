package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"matching-engine/internal/orderbook"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type Handler struct {
	books map[string]*orderbook.OrderBook
	mu    sync.RWMutex
	rdb   *redis.Client
}

func NewHandler(rdb *redis.Client) *Handler {
	return &Handler{
		books: make(map[string]*orderbook.OrderBook),
		rdb:   rdb,
	}
}

func (handler *Handler) Handle(ctx context.Context, event StreamEvent) error {
	switch event.AggregateType {
	case "Order":
		return handler.handleOrder(ctx, event)
	default:
		fmt.Println("unknown aggregate:", event.AggregateType)
	}
	return nil
}

func (handler *Handler) handleOrder(ctx context.Context, event StreamEvent) error {
	switch event.EventType {
	case "order-created":
		var payload struct {
			OrderID  string `json:"order_id"`
			Side     string `json:"side"`
			Type     string `json:"type"`
			Currency string `json:"currency"`
			Price    string `json:"price"`
			Amount   string `json:"amount"`
		}

		if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
			return err
		}

		currency := strings.ToUpper(payload.Currency)
		if currency == "USDT" {
			return fmt.Errorf("invalid currency: USDT is the quote currency and does not have an independent order book")
		}

		handler.mu.RLock()
		book, ok := handler.books[currency]
		handler.mu.RUnlock()

		if !ok {
			handler.mu.Lock()
			book, ok = handler.books[currency]
			if !ok {
				book = orderbook.NewOrderBook(currency, "USDT")
				handler.books[currency] = book
			}
			handler.mu.Unlock()
		}

		price := decimal.Zero
		if payload.Type == "limit" {
			var err error
			price, err = decimal.NewFromString(payload.Price)
			if err != nil {
				return fmt.Errorf("invalid price format: %w", err)
			}
		}

		amount, err := decimal.NewFromString(payload.Amount)
		if err != nil {
			return fmt.Errorf("invalid amount format: %w", err)
		}

		order := orderbook.Order{
			ID:     payload.OrderID,
			Side:   payload.Side,
			Type:   payload.Type,
			Price:  price,
			Amount: amount,
		}

		trades, orderUpdates := book.Match(&order)
		
		if err := handler.cacheOrderBook(ctx, currency, book); err != nil {
			return err
		}

		// Broadcast Order Updates to 'orders-status-stream'
		for _, update := range orderUpdates {
			updateBytes, err := json.Marshal(update)
			if err != nil {
				return fmt.Errorf("failed to marshal order update: %w", err)
			}

			err = handler.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: "orders-status-stream",
				Values: map[string]interface{}{
					"event_type": "order-updated",
					"payload":    string(updateBytes),
				},
			}).Err()

			if err != nil {
				return fmt.Errorf("failed to push order update to redis: %w", err)
			}
		}

		// Broadcast Trade Reports to 'trades-stream' 
		for _, trade := range trades {
			tradeBytes, err := json.Marshal(trade)
			if err != nil {
				return fmt.Errorf("failed to marshal trade: %w", err)
			}

			err = handler.rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: "trades-stream",
				Values: map[string]interface{}{
					"event_type": "trade-executed",
					"payload":    string(tradeBytes),
				},
			}).Err()

			if err != nil {
				return fmt.Errorf("failed to push trade to redis: %w", err)
			}
		}

	default:
		fmt.Println("unknown event:", event.EventType)
	}

	return nil
}

func (handler *Handler) cacheOrderBook(
	ctx context.Context,
	currency string,
	book *orderbook.OrderBook,
) error {
	snapshot := book.Snapshot(10)

	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	return handler.rdb.Set(ctx, "orderbook:"+currency, data, 0).Err()
}

type StreamEvent struct {
	AggregateType string
	EventType     string
	Payload       string
}
