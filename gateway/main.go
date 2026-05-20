package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
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

	ctx := context.Background()
	err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "llm_requests",
		Values: map[string]interface{}{
			"message": body.Message,
		},
	}).Err()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue message"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "queued"})
}