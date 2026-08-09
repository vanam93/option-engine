package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Handlers exposes REST handlers for the Intelligence API.
type Handlers struct {
	cfg  Config
	repo *Repository
}

// NewHandlers creates Intelligence API HTTP handlers.
func NewHandlers(cfg Config, repo *Repository) *Handlers {
	return &Handlers{cfg: cfg.WithDefaults(), repo: repo}
}

func (h *Handlers) parseFilter(c *gin.Context) Filter {
	confidenceMin, _ := strconv.ParseFloat(c.Query("confidence_min"), 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))

	filter := Filter{
		Symbol:        c.Query("symbol"),
		Strategy:      c.Query("strategy"),
		Timeframe:     c.Query("timeframe"),
		Status:        c.Query("status"),
		ConfidenceMin: confidenceMin,
		Limit:         limit,
		Offset:        offset,
		Page:          page,
		Sort:          c.Query("sort"),
		Order:         c.Query("order"),
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			filter.From = t.UTC()
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			filter.To = t.UTC()
		}
	}
	return filter
}

func (h *Handlers) writeError(c *gin.Context, filter Filter, err error, status int) {
	c.JSON(status, Fail(filter, err))
}

func (h *Handlers) ListRecommendations(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	items, pagination, err := h.repo.ListRecommendations(c.Request.Context(), filter)
	recordRequest(time.Since(start), err)
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, OKList(filter, items, pagination))
}

func (h *Handlers) GetRecommendation(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	item, ok, err := h.repo.GetRecommendation(c.Request.Context(), c.Param("id"))
	recordRequest(time.Since(start), errOrNotFound(err, ok))
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	if !ok {
		h.writeError(c, filter, ErrNotFound, http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, OK(filter, item))
}

func (h *Handlers) GetTimeline(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	timeline, ok, err := h.repo.GetTimeline(c.Request.Context(), c.Param("id"))
	recordRequest(time.Since(start), errOrNotFound(err, ok))
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	if !ok {
		h.writeError(c, filter, ErrNotFound, http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, OK(filter, timeline))
}

func (h *Handlers) ListAlerts(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	items, pagination, err := h.repo.ListAlerts(c.Request.Context(), filter)
	recordRequest(time.Since(start), err)
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, OKList(filter, items, pagination))
}

func (h *Handlers) GetOpportunities(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	data, err := h.repo.GetOpportunities(c.Request.Context(), filter)
	recordRequest(time.Since(start), err)
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, OK(filter, data))
}

func (h *Handlers) GetPerformance(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	data, err := h.repo.GetPerformance(c.Request.Context(), filter)
	recordRequest(time.Since(start), err)
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, OK(filter, data))
}

func (h *Handlers) GetOptimization(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	items, pagination, err := h.repo.ListOptimization(c.Request.Context(), filter)
	recordRequest(time.Since(start), err)
	if err != nil {
		h.writeError(c, filter, err, http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, OKList(filter, items, pagination))
}

func (h *Handlers) GetResearch(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	data, err := h.repo.GetResearch(c.Request.Context(), c.Param("id"))
	recordRequest(time.Since(start), err)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrNotFound {
			status = http.StatusNotFound
		}
		h.writeError(c, filter, err, status)
		return
	}
	c.JSON(http.StatusOK, OK(filter, data))
}

func (h *Handlers) GetIntelligenceHealth(c *gin.Context) {
	if !h.cfg.Enabled {
		h.writeError(c, Filter{}, ErrDisabled, http.StatusServiceUnavailable)
		return
	}
	start := time.Now()
	filter := h.parseFilter(c)
	defer func() { recordRequest(time.Since(start), nil) }()
	c.JSON(http.StatusOK, OK(filter, h.repo.IntelligenceHealth()))
}

func errOrNotFound(err error, ok bool) error {
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
