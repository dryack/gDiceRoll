package api

import (
	"context"
	"fmt"
	"github.com/dryack/gDiceRoll/core/session"
	"github.com/jackc/pgx/v4"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/dryack/gDiceRoll/core/admin"
	"github.com/dryack/gDiceRoll/core/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	config         *koanf.Koanf
	router         *gin.Engine
	adminHandler   *admin.AdminHandler
	templates      *template.Template
	cache          *redis.Client
	db             *pgxpool.Pool
	userManager    *user.UserManager
	sessionManager *session.SessionManager
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

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	dbPool, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("Error connecting to Postgres: %v\n", err)
		// Don't return here, allow the server to start without database
	} else {
		fmt.Println("Successfully connected to Postgres")
	}

	userManager := user.NewUserManager(dbPool)
	sessionManager := session.NewSessionManager(cache, dbPool)

	s := &Server{
		config:         cfg,
		router:         router,
		adminHandler:   adminHandler,
		cache:          cache,
		db:             dbPool,
		userManager:    userManager,
		sessionManager: sessionManager,
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

	// User registration and login routes
	s.router.POST("/api/register", s.handleRegister)
	s.router.POST("/api/login", s.handleLogin)
	s.router.POST("/api/logout", s.handleLogout)

	// Admin routes
	s.router.GET("/login", s.adminHandler.LoginPage)
	s.router.POST("/login", s.adminHandler.Login)
	s.router.GET("/dashboard", s.adminHandler.Dashboard)

	log.Println("Routes set up successfully")
}

func (s *Server) handleRegister(c *gin.Context) {
	var registerData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&registerData); err != nil {
		log.Printf("Error binding JSON: %v", err)
		log.Printf("Request body: %v", c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Attempting to create user with username: %s", registerData.Username)

	newUser, err := s.userManager.CreateUser(c.Request.Context(), registerData.Username, registerData.Password, user.User)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	log.Printf("User created successfully with ID: %d", newUser.ID)
	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user_id": newUser.ID})
}

func (s *Server) handleLogin(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		log.Printf("Error binding JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Login attempt for user: %s", loginData.Username)

	user, err := s.userManager.GetUserByUsername(c.Request.Context(), loginData.Username)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("User found: ID=%d, Username=%s, UserType=%s, TwoFactorEnabled=%v",
		user.ID, user.Username, user.UserType, user.TwoFactorEnabled)

	match, err := s.userManager.VerifyPassword(user, loginData.Password)
	if err != nil {
		log.Printf("Error verifying password: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("Password match: %v", match)

	if !match {
		log.Printf("Password does not match for user: %s", loginData.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Create a new session
	session, err := s.sessionManager.CreateSession(c.Request.Context(), user.ID)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	if session == nil {
		log.Printf("Session is nil after creation")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Set session cookie
	c.SetCookie("session_id", session.ID, int(time.Until(session.ExpiresAt).Seconds()), "/", "", false, true)

	log.Printf("Login successful for user: %s, Session ID: %s", loginData.Username, session.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

func (s *Server) handleLogout(c *gin.Context) {
	sessionID, err := c.Cookie("session_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No session found"})
		return
	}

	err = s.sessionManager.DeleteSession(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}

	c.SetCookie("session_id", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}

// AuthMiddleware to check if the user is authenticated
func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		session, err := s.sessionManager.GetSession(c.Request.Context(), sessionID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			c.Abort()
			return
		}

		if time.Now().After(session.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			c.Abort()
			return
		}

		// Set user ID in the context for later use
		c.Set("user_id", session.UserID)
		c.Next()
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

func (s *Server) GetDB() *pgxpool.Pool {
	return s.db
}
