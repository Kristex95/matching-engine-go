package stream

import (
	"context"
	"log"

	"matching-engine/internal/engine"

	"github.com/redis/go-redis/v9"
)

type Consumer struct {
	rdb    *redis.Client
	engine *engine.Handler
}

func NewConsumer(rdb *redis.Client) *Consumer {
	return &Consumer{
		rdb:    rdb,
		engine: engine.NewHandler(),
	}
}

func (c *Consumer) Start() {

	ctx := context.Background()

	for {
		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    "workers",
			Consumer: "consumer-1",
			Streams:  []string{"matching-stream", ">"},
			Count:    10,
			Block:    0,
		}).Result()

		if err != nil {
			log.Fatal(err)
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {

				event := ParseMessage(msg)

				c.engine.Handle(engine.StreamEvent{
					AggregateType: event.AggregateType,
					EventType:     event.EventType,
					Payload:       event.Payload,
				})

				if err != nil {
					log.Println("handler error:", err)
					continue
				}

				err = c.rdb.XAck(ctx, "matching-stream", "workers", msg.ID).Err()
				if err != nil {
					log.Println("ack error:", err)
				}
			}
		}
	}
}
