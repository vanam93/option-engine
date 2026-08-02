package api

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts Intelligence API routes on the given router group.
func RegisterRoutes(group *gin.RouterGroup, cfg Config, repo *Repository) {
	cfg = cfg.withDefaults()
	if !cfg.Enabled || repo == nil {
		return
	}

	group.Use(TimeoutMiddleware(cfg.ReadTimeout))

	h := NewHandlers(cfg, repo)

	group.GET("/recommendations", h.ListRecommendations)
	group.GET("/recommendations/:id/timeline", h.GetTimeline)
	group.GET("/recommendations/:id", h.GetRecommendation)
	group.GET("/alerts", h.ListAlerts)
	group.GET("/opportunities", h.GetOpportunities)
	group.GET("/performance", h.GetPerformance)
	group.GET("/optimization", h.GetOptimization)
	group.GET("/research/:id", h.GetResearch)
	group.GET("/health/intelligence", h.GetIntelligenceHealth)
}
