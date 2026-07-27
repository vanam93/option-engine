package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/option-engine/option-engine/internal/infrastructure/config"
	"github.com/option-engine/option-engine/internal/infrastructure/di"
	"github.com/option-engine/option-engine/internal/infrastructure/logger"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Logging.Level, cfg.Logging.Format)
	log.Info("starting option-engine", "env", cfg.Env)

	ctx := context.Background()
	container, err := di.NewContainer(ctx, cfg, log)
	if err != nil {
		log.Error("failed to initialize", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	// Connect market data provider (config-driven)
	if err := container.ProviderManager.Connect(ctx); err != nil {
		log.Error("provider connect failed", "provider", cfg.Market.Provider, "error", err)
		os.Exit(1)
	}
	log.Info("market provider connected", "provider", cfg.Market.Provider)

	container.HTTPServer.RegisterWebSocket(cfg.WebSocket.Path, func(c *gin.Context) {
		container.WSServer.HandleUpgrade(c.Writer, c.Request)
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- container.HTTPServer.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Shutdown.Timeout)
	defer cancel()

	if err := container.HTTPServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown error", "error", err)
	}

	log.Info("option-engine stopped")
}
