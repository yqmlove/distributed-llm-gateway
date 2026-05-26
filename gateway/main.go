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

	requestID := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pubsub := rdb.Subscribe(ctx, "response:"+requestID)
	defer pubsub.Close()

	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "llm_requests",
		Values: map[string]interface{}{
			"message":    body.Message,
			"request_id": requestID,
		},
	})

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Stream tokens to the client
	for {
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			break
		}
		if msg.Payload == "[DONE]" {
			fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
			c.Writer.Flush()
			break
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", msg.Payload)
		c.Writer.Flush()
	}
}
