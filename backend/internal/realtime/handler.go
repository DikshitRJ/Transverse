package realtime

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"transverse/internal/middleware"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	rdb *redis.Client
}

func NewHandler(rdb *redis.Client) *Handler {
	return &Handler{rdb: rdb}
}

func (h *Handler) StreamEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		// Just a fallback for testing or depending on how the middleware passes it
		userID = r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "Unauthorized or missing user_id", http.StatusUnauthorized)
			return
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	channel := fmt.Sprintf("user:%s:events", userID)
	pubsub := h.rdb.Subscribe(r.Context(), channel)
	defer pubsub.Close()

	// Ensure connection is established
	_, err := pubsub.Receive(r.Context())
	if err != nil {
		log.Printf("Failed to subscribe to redis pubsub: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send an initial connected message
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ch := pubsub.Channel()
	
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return
		case msg := <-ch:
			// msg.Payload is the JSON string from Queue.PublishEvent
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			flusher.Flush()
		}
	}
}
