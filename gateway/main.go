package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	r := gin.Default()
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

	// create unique id
	requestID := uuid.New().String()

	// subscribe the channel and wait for the reply from Worker
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pubsub := rdb.Subscribe(ctx, "response:"+requestID)
	defer pubsub.Close()

	// write message to Redis Stream，attach request_id
	rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "llm_requests",
		Values: map[string]interface{}{
			"message":    body.Message,
			"request_id": requestID,
		},
	})

	// waiting for Worker result （maximum 60 second）
	msg, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "timeout waiting for response"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reply": msg.Payload})
}
