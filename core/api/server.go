package api

import (
	"context"
	"fmt"
	"github.com/dryack/gDiceRoll/core/admin"
	"github.com/dryack/gDiceRoll/core/dsl"
	"github.com/dryack/gDiceRoll/core/jobs"
	"github.com/dryack/gDiceRoll/core/session"
	"github.com/dryack/gDiceRoll/core/user"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
	"html/template"
	"log"
	"time"
)

type Server struct {
	config         *koanf.Koanf
	router         *gin.Engine
	adminHandler   *admin.AdminHandler
	templates      *template.Template
	cache          dsl.Cache
	db             *pgxpool.Pool
	userManager    *user.UserManager
	sessionManager *session.SessionManager
	syncJob        *jobs.SyncJob
	cleanupCtx     context.Context
	cleanupCancel  context.CancelFunc
}

func NewServer(cfg *koanf.Koanf) (*Server, error) {
	router := gin.Default()

	// Load templates
	router.LoadHTMLGlob("web/admin/templates/*")

	// Initialize Dragonfly client
	cacheAddr := cfg.String("dragonfly.address")
	fmt.Printf("Connecting to Dragonfly at %s\n", cacheAddr)
	redisClient := redis.NewClient(&redis.Options{
		Addr: cacheAddr,
	})
	maxCacheEntries := cfg.Int("cache.max_entries")
	if maxCacheEntries <= 0 {
		maxCacheEntries = 100000 // Default value if not set
	}

	// Create DragonflyCache
	cache := dsl.NewDragonflyCache(redisClient, maxCacheEntries)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := redisClient.Ping(ctx).Result()
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

	accessSecret := []byte(cfg.String("jwt.access.secret"))
	refreshSecret := []byte(cfg.String("jwt.refresh.secret"))

	if len(accessSecret) == 0 || len(refreshSecret) == 0 {
		return nil, fmt.Errorf("JWT secrets are not properly configured")
	}

	userManager := user.NewUserManager(dbPool)
	sessionManager, err := session.NewSessionManager(redisClient, dbPool, cfg.String("jwt.access.secret"), cfg.String("jwt.refresh.secret"))
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %v", err)
	}

	adminHandler, err := admin.NewAdminHandler(userManager, sessionManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin handler: %v", err)
	}

	syncJob := jobs.NewSyncJob(cache, dbPool, maxCacheEntries)
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())

	s := &Server{
		config:         cfg,
		router:         router,
		adminHandler:   adminHandler,
		cache:          cache,
		db:             dbPool,
		userManager:    userManager,
		sessionManager: sessionManager,
		syncJob:        syncJob,
		cleanupCtx:     cleanupCtx,
		cleanupCancel:  cleanupCancel,
	}
	s.setupRoutes()
	// Start the cleanup task
	s.sessionManager.StartCleanupTask(cleanupCtx, 1*time.Hour) // Run cleanup every hour
	// Start the sync job
	go s.syncJob.Start(cleanupCtx)

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
	// Dice handling
	s.router.GET("/api/roll", s.handleDiceRoll)
	s.router.GET("/api/encode", s.handleEncodeExpression)
	// Admin routes
	adminGroup := s.router.Group("/admin")
	{
		adminGroup.GET("/login", s.adminHandler.LoginPage)
		adminGroup.POST("/login", s.adminHandler.Login)
		adminGroup.GET("/dashboard", s.AuthMiddleware(), s.adminHandler.Dashboard)
	}

	log.Println("Routes set up successfully")
}

func (s *Server) setCacheAndLog(key, value string) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := s.cache.SetGeneral(ctx, key, value, 30*time.Second)
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
	defer s.cleanupCancel() // Ensure cleanup task is stopped when server stops

	log.Printf("Starting server on %s", s.config.String("server.address"))
	return s.router.Run(s.config.String("server.address"))
}

func (s *Server) GetDB() *pgxpool.Pool {
	return s.db
}
