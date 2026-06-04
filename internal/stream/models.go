package stream

import "github.com/redis/go-redis/v9"

type StreamMessage struct {
	ID            string
	AggregateType string
	AggregateID   string
	EventType     string
	Payload       string
}

type RawMessage = redis.XMessage