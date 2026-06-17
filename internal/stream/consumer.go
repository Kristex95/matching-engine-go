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
		engine: engine.NewHandler(rdb),
	}
}

func (c *Consumer) Start() {
	ctx := context.Background()
	streamName := "matching-stream"
	groupName := "workers"

	err := c.rdb.XGroupCreateMkStream(ctx, streamName, groupName, "0").Err()
	if err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			log.Fatalf("failed to initialize stream/group: %v", err)
		}
	}

	for {
		// 2. Now you can safely read from the group
		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    groupName,
			Consumer: "consumer-1",
			Streams:  []string{streamName, ">"},
			Count:    10,
			Block:    0,
		}).Result()

		if err != nil {
			log.Println("read error:", err)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				event := ParseMessage(msg)

				err = c.engine.Handle(ctx, engine.StreamEvent{
					AggregateType: event.AggregateType,
					EventType:     event.EventType,
					Payload:       event.Payload,
				})

				if err != nil {
					log.Println("handler error:", err)
					continue
				}

				err = c.rdb.XAck(ctx, streamName, groupName, msg.ID).Err()
				if err != nil {
					log.Println("ack error:", err)
				}
			}
		}
	}
}
