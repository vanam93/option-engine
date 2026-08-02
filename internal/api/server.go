package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// Server owns Intelligence API route registration and middleware.
type Server struct {
	cfg  Config
	repo *Repository
}

// NewServer creates an Intelligence API server facade.
func NewServer(cfg Config, repo *Repository) (*Server, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, fmt.Errorf("api: nil repository")
	}
	return &Server{cfg: cfg, repo: repo}, nil
}

// Enabled reports whether the Intelligence API is active.
func (s *Server) Enabled() bool {
	return s.cfg.Enabled
}

// Prefix returns the configured API route prefix.
func (s *Server) Prefix() string {
	return s.cfg.Prefix
}

// Register mounts routes on the given Gin engine.
func (s *Server) Register(engine *gin.Engine) {
	if !s.cfg.Enabled {
		return
	}
	RegisterRoutes(engine.Group(s.cfg.Prefix), s.cfg, s.repo)
}

// RegisterGroup mounts routes on an existing router group.
func (s *Server) RegisterGroup(group *gin.RouterGroup) {
	RegisterRoutes(group, s.cfg, s.repo)
}

// Config returns the resolved server configuration.
func (s *Server) Config() Config {
	return s.cfg
}

// Repository returns the read-only repository.
func (s *Server) Repository() *Repository {
	return s.repo
}
