package main

import (
	"github.com/redis/go-redis/v9"
	"matching-engine/internal/stream"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	consumer := stream.NewConsumer(rdb)
	consumer.Start()
}