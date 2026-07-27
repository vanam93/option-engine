package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "github.com/option-engine/option-engine/internal/adapters/http"
	"github.com/option-engine/option-engine/internal/infrastructure/config"
	"github.com/option-engine/option-engine/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := &config.Config{Env: "test"}
	log := logger.New("error", "text")
	srv := httpserver.NewServer(cfg, log, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestStatusEndpoint(t *testing.T) {
	cfg := &config.Config{Env: "test"}
	log := logger.New("error", "text")
	srv := httpserver.NewServer(cfg, log, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "option-engine", body["service"])
}
