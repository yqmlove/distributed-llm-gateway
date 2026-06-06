package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func main() {
	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"POST", "GET"},
		AllowHeaders: []string{"Content-Type"},
	}))

	r.OPTIONS("/chat", func(c *gin.Context) { c.Status(200) })
	r.POST("/chat", handleChat)
	r.Run(":8080")
}

func handleChat(c *gin.Context) {
	var body struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	// Generate a unique ID for this request
	requestID := uuid.New().String()
	fmt.Printf("[%s] request received: %s\n", requestID, body.Message)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Subscribe before writing to the stream to avoid missing the first token
	pubsub := rdb.Subscribe(ctx, "response:"+requestID)
	defer pubsub.Close()

	// Write the message into the Redis Stream for the worker to pick up
	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "llm_requests",
		Values: map[string]interface{}{
			"message":    body.Message,
			"request_id": requestID,
		},
	})
	fmt.Printf("[%s] message queued to Redis Stream\n", requestID)

	// Set SSE headers so the browser knows this is a stream
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Forward each token to the client as it arrives
	tokenCount := 0
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			fmt.Printf("[%s] stream error: %v\n", requestID, err)
			break
		}
		// [DONE] is the end marker sent by the worker when streaming is complete
		if msg.Payload == "[DONE]" {
			fmt.Printf("[%s] stream complete, tokens sent: %d\n", requestID, tokenCount)
			fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
			c.Writer.Flush()
			break
		}
		tokenCount++
		fmt.Fprintf(c.Writer, "data: %s\n\n", msg.Payload)
		c.Writer.Flush()
	}
}
