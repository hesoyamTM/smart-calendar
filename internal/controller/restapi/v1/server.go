// Package v1 is the REST API for the application.
package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	claude "github.com/hesoyamTM/smart-calendar/internal/adapters/clients/claude"
	google "github.com/hesoyamTM/smart-calendar/internal/adapters/clients/google"
)

const (
	cookieGoogle = "google_token"
	stateExpiry  = 10 * time.Minute
	cookieMaxAge = 900 // 15 minutes
)

type stateEntry struct {
	expiry time.Time
}

type Server struct {
	cfg       Config
	gcCfg     google.Config
	claudeCfg claude.Config
	logger    *slog.Logger
	upgrader  websocket.Upgrader

	mu           sync.Mutex
	googleStates map[string]stateEntry
}

func New(cfg Config, gcCfg google.Config, claudeCfg claude.Config, logger *slog.Logger) *Server {
	return &Server{
		cfg:          cfg,
		gcCfg:        gcCfg,
		claudeCfg:    claudeCfg,
		logger:       logger,
		googleStates: make(map[string]stateEntry),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (c *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", c.handleIndex)
	mux.HandleFunc("/status", c.handleStatus)
	mux.HandleFunc("/auth/google", c.handleGoogleAuth)
	mux.HandleFunc("/oauth2callback", c.handleGoogleCallback)
	mux.HandleFunc("/ws", c.handleWS)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", c.cfg.Port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	base := fmt.Sprintf("http://localhost:%d", c.cfg.Port)
	c.logger.Info("server started", "port", c.cfg.Port)
	c.logger.Info("open in browser", "url", base)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}
