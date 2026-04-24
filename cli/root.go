// Package cli defines the Cobra command tree for the smart-calendar binary.
package cli

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hesoyamTM/smart-calendar/internal/application"
	"github.com/hesoyamTM/smart-calendar/internal/config"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	var (
		cfgFile string
		port    int
	)

	cmd := &cobra.Command{
		Use:   "smart-calendar",
		Short: "AI-powered smart calendar assistant",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("port") {
				cfg.HTTP.Port = port
			}

			app := application.New(cfg, logger)

			ctx, stop := signal.NotifyContext(
				context.Background(),
				os.Interrupt,
				syscall.SIGTERM,
				syscall.SIGINT,
			)
			defer stop()

			return app.Run(ctx)
		},
	}

	cmd.Flags().StringVarP(&cfgFile, "config", "c", "config.yaml", "path to YAML config file")
	cmd.Flags().IntVarP(&port, "port", "p", 0, "HTTP server port (overrides config)")

	return cmd
}
