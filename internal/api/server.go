package api

import (
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

type Server struct {
	rdb *redis.Client
}

func NewServer(rdb *redis.Client) *Server {
	return &Server{rdb: rdb}
}

func (s *Server) HandleGetOrderBook(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[1] == "" {
		http.Error(w, `{"error": "currency parameter is required"}`, http.StatusBadRequest)
		return
	}

	currency := strings.ToUpper(pathParts[1])
	ctx := r.Context()

	val, err := s.rdb.Get(ctx, "orderbook:"+currency).Result()
	if err == redis.Nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": "order book not found for this currency"}`))
		return
	} else if err != nil {
		http.Error(w, `{"error": "failed to fetch order book"}`, http.StatusInternalServerError)
		return
	}

	// The stored data is already raw JSON stringified from your Snapshot
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(val))
}