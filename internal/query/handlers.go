package query

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handlers exposes REST handlers for the query API.
type Handlers struct {
	api *API
}

// NewHandlers creates query HTTP handlers.
func NewHandlers(api *API) *Handlers {
	return &Handlers{api: api}
}

func (h *Handlers) parseFilter(c *gin.Context) Filter {
	confidenceMin, _ := strconv.ParseFloat(c.Query("confidence_min"), 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	return Filter{
		Symbol:        c.Query("symbol"),
		Strategy:      c.Query("strategy"),
		Timeframe:     c.Query("timeframe"),
		Status:        c.Query("status"),
		ConfidenceMin: confidenceMin,
		Limit:         limit,
		Offset:        offset,
	}
}

func (h *Handlers) writeError(c *gin.Context, err error) {
	switch err {
	case ErrDisabled:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	case ErrNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

// ListRecommendations handles GET /recommendations.
func (h *Handlers) ListRecommendations(c *gin.Context) {
	resp, err := h.api.ListRecommendations(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetRecommendation handles GET /recommendations/:id.
func (h *Handlers) GetRecommendation(c *gin.Context) {
	filter := h.parseFilter(c)
	resp, err := h.api.GetRecommendation(c.Param("id"), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetTimeline handles GET /recommendations/:id/timeline.
func (h *Handlers) GetTimeline(c *gin.Context) {
	filter := h.parseFilter(c)
	resp, err := h.api.GetTimeline(c.Param("id"), filter)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ListAlerts handles GET /alerts.
func (h *Handlers) ListAlerts(c *gin.Context) {
	resp, err := h.api.ListAlerts(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetOpportunities handles GET /opportunities.
func (h *Handlers) GetOpportunities(c *gin.Context) {
	resp, err := h.api.GetOpportunities(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetScanner handles GET /scanner.
func (h *Handlers) GetScanner(c *gin.Context) {
	resp, err := h.api.GetScanner(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetPerformance handles GET /performance.
func (h *Handlers) GetPerformance(c *gin.Context) {
	resp, err := h.api.GetPerformance(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetOptimization handles GET /optimization.
func (h *Handlers) GetOptimization(c *gin.Context) {
	resp, err := h.api.GetOptimization(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetResearch handles GET /research/:id.
func (h *Handlers) GetResearch(c *gin.Context) {
	resp, err := h.api.GetResearch(c.Request.Context(), c.Param("id"), h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetIntelligenceHealth handles GET /health/intelligence.
func (h *Handlers) GetIntelligenceHealth(c *gin.Context) {
	resp, err := h.api.GetIntelligenceHealth(h.parseFilter(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
