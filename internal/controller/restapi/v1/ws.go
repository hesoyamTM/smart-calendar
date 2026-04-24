package v1

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	claude "github.com/hesoyamTM/smart-calendar/internal/adapters/clients/claude"
	google "github.com/hesoyamTM/smart-calendar/internal/adapters/clients/google"
	"github.com/hesoyamTM/smart-calendar/internal/models"
	"github.com/hesoyamTM/smart-calendar/internal/service"
)

func (c *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	googleTok, err := tokenFromCookie(r, cookieGoogle)
	if err != nil {
		http.Error(w, "Google not authenticated — visit /auth/google first", http.StatusUnauthorized)
		return
	}

	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {
		c.logger.Error("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	c.logger.Info("websocket session started", "remote", r.RemoteAddr)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	calClient, err := google.New(ctx, c.gcCfg, googleTok)
	if err != nil {
		c.logger.Error("google client init failed", "error", err)
		conn.WriteMessage(
			websocket.TextMessage,
			chunkFrame(models.Chunk{Text: "error: failed to connect to Google Calendar"}),
		)
		return
	}

	llm := claude.New(c.claudeCfg)
	svc := service.NewSmartCalendarService(c.logger, llm, calClient)
	inputCh := make(chan string)

	defer close(inputCh)

	outputCh := svc.Execute(ctx, inputCh)

	go func() {
		for chunk := range outputCh {
			frame := chunkFrame(chunk)
			if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
				c.logger.Error("ws write error", "error", err)
				cancel()
				return
			}
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Error("ws read error", "error", err)
			}
			cancel()
			break
		}
		select {
		case inputCh <- string(raw):
		case <-ctx.Done():
			return
		}
	}

	c.logger.Info("websocket session ended", "remote", r.RemoteAddr)
}
