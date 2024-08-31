package api

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/dryack/gDiceRoll/core/admin"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	config       *koanf.Koanf
	router       *gin.Engine
	adminHandler *admin.AdminHandler
	templates    *template.Template
	cache        *redis.Client
	db           *pgxpool.Pool
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

	// Initialize Postgres connection
	dbHost := cfg.String("postgres.host")
	dbPort := cfg.String("postgres.port")
	dbUser := cfg.String("postgres.user")
	dbPassword := cfg.String("postgres.password")
	dbName := cfg.String("postgres.dbname")

	// DEBUG
	// fmt.Printf("Debug: postgres.host=%s\n", dbHost)
	// fmt.Printf("Debug: postgres.port=%s\n", dbPort)
	// fmt.Printf("Debug: postgres.user=%s\n", dbUser)
	// fmt.Printf("Debug: postgres.dbname=%s\n", dbName)
	// Don't print the password for security reasons

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	// fmt.Printf("Connecting to Postgres at postgres://%s:****@%s:%s/%s?sslmode=disable\n", dbUser, dbHost, dbPort, dbName) // DEBUG

	dbPool, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Error connecting to Postgres: %v\n", err)
		// Don't return here, allow the server to start without database
	} else {
		fmt.Println("Successfully connected to Postgres")
	}

	s := &Server{
		config:       cfg,
		router:       router,
		adminHandler: adminHandler,
		cache:        cache,
		db:           dbPool,
	}
	s.setupRoutes()
	return s, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(func(c *gin.Context) {
		log.Printf("Incoming request: %s %s", c.Request.Method, c.Request.URL.Path)
		for name, values := range c.Request.Header {
			log.Printf("Header %s: %v", name, values)
		}
		c.Next()
	})

	s.router.GET("/api/hello", s.handleHello)

	// Admin routes
	s.router.GET("/login", s.adminHandler.LoginPage)
	s.router.POST("/login", s.adminHandler.Login)
	s.router.GET("/dashboard", s.adminHandler.Dashboard)

	log.Println("Routes set up successfully")
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

func (s *Server) setCacheAndLog(key, value string) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := s.cache.Set(ctx, key, value, 30*time.Second).Err()
	if err != nil {
		fmt.Printf("Failed to set cache: %v\n", err)
	}
}

func (s *Server) setCacheAndDatabase(key, value string) {
	s.setCacheAndLog(key, value)

	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_, err := s.db.Exec(ctx, "INSERT INTO hello_messages (key, message) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET message = $2", key, value)
		if err != nil {
			fmt.Printf("Failed to set database: %v\n", err)
		}
	}
}

func (s *Server) Run() error {
	log.Printf("Starting server on %s", s.config.String("server.address"))
	return s.router.Run(s.config.String("server.address"))
}
