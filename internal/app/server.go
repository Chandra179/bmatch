package app

import (
	"context"
	"fmt"
	"log"

	"bmatch/cfg"
	"bmatch/internal/service/auth"
	"bmatch/internal/service/group"
	"bmatch/internal/service/user"
	"bmatch/pkg/cache"
	"bmatch/pkg/db"
	"bmatch/pkg/logger"
	"bmatch/pkg/oauth2"
	"bmatch/pkg/session"

	"github.com/gin-gonic/gin"
)

// Server holds all application dependencies
type Server struct {
	config        *cfg.Config
	router        *gin.Engine
	logger        *logger.AppLogger
	db            *db.SQLClient
	cache         cache.Cache
	sessionStore  session.Store
	oauth2Manager *oauth2.Manager
	shutdown      func(context.Context) error

	// internal service
	userService  *user.Service
	groupService *group.Service
	authService  *auth.Service
}

// NewServer creates and initializes a new server instance
func NewServer(ctx context.Context, config *cfg.Config) (*Server, error) {
	s := &Server{
		config: config,
	}

	shutdown, err := setupObservability(ctx, &config.Observability)
	if err != nil {
		return nil, fmt.Errorf("observability setup: %w", err)
	}
	s.shutdown = shutdown

	s.logger = logger.NewLogger(config.AppEnv)
	s.logger.Info(ctx, "Initializing server...")

	if err := s.initDatabase(); err != nil {
		return nil, fmt.Errorf("database init: %w", err)
	}

	if err := s.initCache(); err != nil {
		return nil, fmt.Errorf("cache init: %w", err)
	}

	s.sessionStore = session.NewInMemoryStore()

	if err := s.initOAuth2(ctx); err != nil {
		return nil, fmt.Errorf("oauth2 init: %w", err)
	}

	s.initServicesAndRoutes()

	s.logger.Info(ctx, "Server initialized successfully")
	return s, nil
}

func (s *Server) initDatabase() error {
	pg := s.config.Postgres
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		pg.User, pg.Password, pg.Host, pg.Port, pg.DBName, pg.SSLMode,
	)

	dbClient, err := db.NewSQLClient("postgres", dsn)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	s.db = dbClient

	if err := runMigrations(dsn); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}

	return nil
}

func (s *Server) initCache() error {
	addr := s.config.Redis.Host + ":" + s.config.Redis.Port
	s.cache = cache.NewRedisCache(addr)
	return nil
}

func (s *Server) initOAuth2(ctx context.Context) error {
	mgr, err := oauth2.NewManager(ctx, &s.config.OAuth2)
	if err != nil {
		return err
	}
	s.oauth2Manager = mgr
	return nil
}

func (s *Server) initServicesAndRoutes() {
	s.authService = auth.NewService(
		s.oauth2Manager,
		s.sessionStore,
		s.db,
	)

	// Wire up OAuth2 callback
	s.oauth2Manager.CallbackHandler = func(
		ctx context.Context,
		provider string,
		userInfo *oauth2.UserInfo,
		tokenSet *oauth2.TokenSet,
	) (*oauth2.CallbackInfo, error) {
		return s.authService.HandleCallback(ctx, provider, userInfo, tokenSet)
	}

	// Initialize User Service
	userRepo := user.NewRepository(s.db)
	s.userService = user.NewService(userRepo, s.logger)
	// Initialize Group Service
	groupRepo := group.NewRepository(s.db)
	groupMatcher := group.NewPostgresMatcher(groupRepo)
	s.groupService = group.NewService(groupRepo, groupMatcher, s.cache, s.logger)

	r := gin.New()
	r.Use(gin.Recovery())
	routes := NewRoutes(r)
	routes.setupInfraRoutes()
	// Business logic endpoints
	authHandler := auth.NewHandler(s.authService)
	routes.setupAuthRoutes(authHandler, s.oauth2Manager)
	routes.setupGroupRoutes(authHandler, s.groupService)
	routes.setupUserRoutes(authHandler, s.userService)

	s.router = r
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	log.Printf("Server listening on %s", addr)
	return s.router.Run(addr)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.shutdown != nil {
		if err := s.shutdown(ctx); err != nil {
			return fmt.Errorf("observability shutdown: %w", err)
		}
	}
	return nil
}
