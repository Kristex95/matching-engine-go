package engine

import (
	"encoding/json"
	"fmt"
	"strconv"

	"matching-engine/internal/orderbook"
)

type Handler struct {
	book *orderbook.OrderBook
}

func NewHandler() *Handler {
	return &Handler{
		book: orderbook.NewOrderBook(),
	}
}

func (handler *Handler) Handle(event StreamEvent) error {

	switch event.AggregateType {

	case "Order":
		return handler.handleOrder(event)

	default:
		fmt.Println("unknown aggregate:", event.AggregateType)
	}

	return nil
}

func (handler *Handler) handleOrder(event StreamEvent) error {

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

		price, err := strconv.ParseFloat(payload.Price, 64)
		if err != nil {
			return err
		}

		amount, err := strconv.ParseFloat(payload.Amount, 64)
		if err != nil {
			return err
		}

		order := orderbook.Order{
			ID:     payload.OrderID,
			Side:   payload.Side,
			Type:   payload.Type,
			Price:  price,
			Amount: amount,
		}
		
		handler.book.Match(&order)
		handler.book.Print()

	default:
		fmt.Println("unknown event:", event.EventType)
	}

	return nil
}

type StreamEvent struct {
	AggregateType string
	EventType     string
	Payload       string
}
