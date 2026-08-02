package ws_test

import (
	"testing"

	"github.com/vanam-gangireddy/option-engine/internal/adapters/ws"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/config"
	"github.com/vanam-gangireddy/option-engine/internal/infrastructure/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewHub(t *testing.T) {
	cfg := &config.Config{}
	cfg.WebSocket.ReadBufferSize = 1024
	cfg.WebSocket.WriteBufferSize = 1024

	log := logger.New("error", "text")
	hub := ws.NewHub(cfg, log)

	assert.Equal(t, 0, hub.ClientCount())
}

func TestBroadcastNoClients(t *testing.T) {
	cfg := &config.Config{}
	log := logger.New("error", "text")
	hub := ws.NewHub(cfg, log)

	// Should not panic with zero clients
	hub.Broadcast([]byte(`{"type":"test"}`))
	assert.Equal(t, 0, hub.ClientCount())
}
