package query

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts query API routes on the given router group.
func RegisterRoutes(group *gin.RouterGroup, api *API) {
	if api == nil || !api.enabled() {
		return
	}
	h := NewHandlers(api)

	group.GET("/recommendations", h.ListRecommendations)
	group.GET("/recommendations/:id/timeline", h.GetTimeline)
	group.GET("/recommendations/:id", h.GetRecommendation)
	group.GET("/alerts", h.ListAlerts)
	group.GET("/opportunities", h.GetOpportunities)
	group.GET("/scanner", h.GetScanner)
	group.GET("/performance", h.GetPerformance)
	group.GET("/optimization", h.GetOptimization)
	group.GET("/research/:id", h.GetResearch)
	group.GET("/health/intelligence", h.GetIntelligenceHealth)
}
