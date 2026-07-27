package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/option-engine/option-engine/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(cfgFile, []byte(`
env: test
http:
  port: 9090
postgres:
  host: db
`), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(cfgFile)
	require.NoError(t, err)

	assert.Equal(t, "test", cfg.Env)
	assert.Equal(t, 9090, cfg.HTTP.Port)
	assert.Equal(t, "db", cfg.Postgres.Host)
	assert.Equal(t, "0.0.0.0", cfg.HTTP.Host) // default
}

func TestHTTPAddr(t *testing.T) {
	cfg := &config.Config{}
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 8080
	assert.Equal(t, "127.0.0.1:8080", cfg.HTTPAddr())
}

func TestPostgresDSN(t *testing.T) {
	cfg := &config.Config{}
	cfg.Postgres.User = "user"
	cfg.Postgres.Password = "pass"
	cfg.Postgres.Host = "localhost"
	cfg.Postgres.Port = 5432
	cfg.Postgres.Database = "db"
	cfg.Postgres.SSLMode = "disable"

	dsn := cfg.PostgresDSN()
	assert.Contains(t, dsn, "postgres://user:pass@localhost:5432/db")
}
