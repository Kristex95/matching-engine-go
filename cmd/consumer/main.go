package main

import (
	"context"
	"log"
	"net/http"

	"matching-engine/internal/api"
	"matching-engine/internal/stream"

	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	consumer := stream.NewConsumer(rdb)
	go func() {
		log.Println("Starting stream consumer...")
		consumer.Start() 
	}()

	apiServer := api.NewServer(rdb)
	
	http.HandleFunc("/orderbook/", apiServer.HandleGetOrderBook)

	log.Println("HTTP Server listening on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}