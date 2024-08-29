package api

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/dryack/gDiceRoll/core/admin"
	"github.com/gin-gonic/gin"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	config       *koanf.Koanf
	router       *gin.Engine
	adminHandler *admin.AdminHandler
	templates    *template.Template
	cache        *redis.Client
}

func NewServer(cfg *koanf.Koanf) (*Server, error) {
	adminHandler, err := admin.NewAdminHandler()
	if err != nil {
		return nil, err
	}

	router := gin.Default()

	// Load templates
	router.LoadHTMLGlob("web/admin/templates/*")

	// Initialize Dragonfly client
	cacheAddr := cfg.String("dragonfly.address")
	fmt.Printf("Connecting to Dragonfly at %s\n", cacheAddr)
	cache := redis.NewClient(&redis.Options{
		Addr: cacheAddr,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = cache.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("Error connecting to Dragonfly: %v\n", err)
		// Don't return here, allow the server to start without cache
	} else {
		fmt.Println("Successfully connected to Dragonfly")
	}

	s := &Server{
		config:       cfg,
		router:       router,
		adminHandler: adminHandler,
		cache:        cache,
	}
	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	s.router.GET("/api/hello", s.handleHello)

	// Admin routes
	adminGroup := s.router.Group("/admin")
	{
		adminGroup.GET("/login", s.adminHandler.LoginPage)
		adminGroup.POST("/login", s.adminHandler.Login)
		adminGroup.GET("/dashboard", s.adminHandler.Dashboard)
	}
}

func (s *Server) handleHello(c *gin.Context) {
	message := c.Query("message")
	if message == "" {
		message = "Hello, World!"
	}

	cacheKey := fmt.Sprintf("hello:%s", message)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 350*time.Millisecond)
	defer cancel()

	// Try to get the message from cache
	cachedMessage, err := s.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		// Cache hit
		c.JSON(http.StatusOK, gin.H{"message": cachedMessage, "source": "cache"})
		return
	}

	if err != redis.Nil {
		// Log the error, but continue with the request
		fmt.Printf("Cache error: %v\n", err)
	}

	// Cache miss or error, set the cache
	err = s.cache.Set(ctx, cacheKey, message, 30*time.Second).Err()
	if err != nil {
		// Log the error, but continue with the request
		fmt.Printf("Failed to set cache: %v\n", err)
	}

	// Return the message
	c.JSON(http.StatusOK, gin.H{"message": message, "source": "direct"})
}

func (s *Server) Run() error {
	return s.router.Run(s.config.String("server.address"))
}
