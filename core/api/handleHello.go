package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/redis/go-redis/v9"
	"net/http"
	"time"
)

func (s *Server) handleHello(c *gin.Context) {
	message := c.Query("message")
	if message == "" {
		message = "Hello, World!"
	}

	cacheKey := fmt.Sprintf("hello:%s", message)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 350*time.Millisecond)
	defer cancel()

	// Try to get the message from cache
	cachedMessage, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache hit
		c.JSON(http.StatusOK, gin.H{"message": cachedMessage, "source": "cache"})
		return
	}

	if !errors.Is(err, redis.Nil) {
		// Log the cache error, but continue with the request
		fmt.Printf("Cache error: %v\n", err)
	}

	// Cache miss, try to get from database
	var dbMessage string
	if s.db != nil {
		err = s.db.QueryRow(ctx, "SELECT message FROM hello_messages WHERE key = $1", cacheKey).Scan(&dbMessage)
		if err == nil {
			// Database hit
			c.JSON(http.StatusOK, gin.H{"message": dbMessage, "source": "database"})
			// Set cache
			go s.setCacheAndLog(cacheKey, dbMessage)
			return
		}
		if err != pgx.ErrNoRows {
			// Log the database error, but continue with the request
			fmt.Printf("Database error: %v\n", err)
		}
	}

	// Cache and database miss, or errors
	c.JSON(http.StatusOK, gin.H{"message": message, "source": "direct"})

	// Set cache and database asynchronously
	go s.setCacheAndDatabase(cacheKey, message)
}
