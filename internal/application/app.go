// Package application wires all modules together and owns the top-level config.
package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hesoyamTM/smart-calendar/internal/config"
	v1 "github.com/hesoyamTM/smart-calendar/internal/controller/restapi/v1"
)

type App struct {
	server *v1.Server
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) *App {
	cfg.SetDefaults()

	cfg.Google.RedirectURL = fmt.Sprintf("http://localhost:%d/oauth2callback", cfg.HTTP.Port)

	server := v1.New(
		v1.Config{Port: cfg.HTTP.Port},
		cfg.Google,
		cfg.Claude,
		logger,
	)

	return &App{server: server, logger: logger}
}

func (a *App) Run(ctx context.Context) error {
	return a.server.Run(ctx)
}
