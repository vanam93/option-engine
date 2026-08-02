package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vanam-gangireddy/option-engine/internal/application/ports"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/config"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/logger"
)

// Server wraps the Gin HTTP server.
type Server struct {
	engine *gin.Engine
	server *http.Server
	log    *slog.Logger
}

// NewServer creates the HTTP server with routes and middleware.
func NewServer(cfg *config.Config, log *slog.Logger, healthCheckers []ports.HealthChecker, healthReporters []ports.HealthReporter) *Server {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestIDMiddleware())
	engine.Use(requestLogger(log))

	registerRoutes(engine, cfg, healthCheckers, healthReporters)

	s := &Server{
		engine: engine,
		log:    log,
		server: &http.Server{
			Addr:              cfg.HTTPAddr(),
			Handler:           engine,
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
	return s
}

func registerRoutes(engine *gin.Engine, cfg *config.Config, healthCheckers []ports.HealthChecker, healthReporters []ports.HealthReporter) {
	engine.GET("/health", healthHandler(healthCheckers, false))
	engine.GET("/ready", healthHandler(healthCheckers, true))
	engine.GET("/health/components", componentsHealthHandler(healthReporters))

	v1 := engine.Group("/api/v1")
	{
		v1.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":  "option-engine",
				"version":  "0.2.0",
				"env":      cfg.Env,
				"provider": cfg.Market.Provider,
			})
		})
	}
}

func healthHandler(checkers []ports.HealthChecker, deep bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deep {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()

			for _, checker := range checkers {
				if err := checker.Check(ctx); err != nil {
					c.JSON(http.StatusServiceUnavailable, gin.H{
						"status": "not_ready",
						"error":  err.Error(),
					})
					return
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

func componentsHealthHandler(reporters []ports.HealthReporter) gin.HandlerFunc {
	return func(c *gin.Context) {
		reports := make([]any, 0, len(reporters))
		for _, r := range reporters {
			reports = append(reports, r.Health())
		}
		c.JSON(http.StatusOK, gin.H{"components": reports})
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.New().String()
		}
		cid := c.GetHeader("X-Correlation-ID")
		if cid == "" {
			cid = rid
		}
		ctx := logger.ContextWithRequestID(c.Request.Context(), rid)
		ctx = logger.ContextWithCorrelationID(ctx, cid)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Request-ID", rid)
		c.Header("X-Correlation-ID", cid)
		c.Next()
	}
}

func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		logger.FromContext(c.Request.Context(), log).Info("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latency.Milliseconds(),
		)
	}
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	s.log.Info("starting http server", "addr", s.server.Addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Handler returns the underlying http.Handler (for testing).
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

// RegisterWebSocket attaches a WebSocket handler at the given path.
func (s *Server) RegisterWebSocket(path string, handler gin.HandlerFunc) {
	s.engine.GET(path, handler)
}

// Engine returns the Gin engine for advanced route registration.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}
