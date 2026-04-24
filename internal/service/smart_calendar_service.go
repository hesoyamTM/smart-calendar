// Package service provides the business logic for the smart calendar application.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hesoyamTM/smart-calendar/internal/models"
)

const systemPrompt = `You are a smart calendar assistant. You help the user manage ` +
	`their Calendar: create, update, delete, list, and get events, list ` +
	`calendars, respond to invitations, and suggest meeting times. ` +
	`Use the provided tools to act on the calendar. Ask the user clarifying ` +
	`questions only when information is missing or ambiguous. Prefer to look up ` +
	`state (list_calendars, list_events) before asking.`

type SmartCalendarService struct {
	logger   *slog.Logger
	llm      LLMClient
	calendar CalendarClient
}

func NewSmartCalendarService(
	logger *slog.Logger,
	llm LLMClient,
	calendar CalendarClient,
) *SmartCalendarService {
	return &SmartCalendarService{
		logger:   logger,
		llm:      llm,
		calendar: calendar,
	}
}

// Execute runs the agent loop. It reads user inputs from inputCh and writes
// streaming chunks to the returned channel. Each response ends with a Chunk{Done: true}.
func (s *SmartCalendarService) Execute(ctx context.Context, inputCh <-chan string) chan models.Chunk {
	outputCh := make(chan models.Chunk)

	go func() {
		defer close(outputCh)

		history := []models.Message{{Role: models.RoleSystem, Content: systemPrompt}}

		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					s.logger.Error("context error", "error", ctx.Err())
				}
				s.logger.Info("context done")
				return
			case input, ok := <-inputCh:
				if !ok {
					s.logger.Info("input channel closed")
					return
				}

				history = append(history, models.Message{Role: models.RoleUser, Content: input})

				if err := s.runTurn(ctx, &history, inputCh, outputCh); err != nil {
					if ctx.Err() != nil {
						return
					}
					s.logger.Error("agent turn failed", "error", err)
					if !sendChunk(ctx, outputCh, fmt.Sprintf("Sorry, I hit an error: %v", err)) {
						return
					}
					sendDone(ctx, outputCh)
				}
			}
		}
	}()

	return outputCh
}

// runTurn drives the LLM until it produces a final answer for the current user input.
// It may ask clarifying questions and invoke tools as needed.
func (s *SmartCalendarService) runTurn(
	ctx context.Context,
	history *[]models.Message,
	inputCh <-chan string,
	outputCh chan<- models.Chunk,
) error {
	const maxSteps = 20

	for range maxSteps {
		streamEventCh := s.llm.NextAction(ctx, *history)
		action, err := s.readStream(ctx, streamEventCh, outputCh)
		if err != nil {
			return fmt.Errorf("llm: %w", err)
		}

		switch {
		case action.FinalAnswer != "":
			*history = append(*history, models.Message{Role: models.RoleAssistant, Content: action.FinalAnswer})
			if !sendDone(ctx, outputCh) {
				return ctx.Err()
			}
			return nil

		case action.ClarifyingQuestion != "":
			*history = append(*history, models.Message{Role: models.RoleAssistant, Content: action.ClarifyingQuestion})
			if !sendDone(ctx, outputCh) {
				return ctx.Err()
			}
			answer, ok := recv(ctx, inputCh)
			if !ok {
				return ctx.Err()
			}
			*history = append(*history, models.Message{Role: models.RoleUser, Content: answer})

		case len(action.ToolCalls) > 0:
			*history = append(*history, models.Message{
				Role:      models.RoleAssistant,
				ToolCalls: action.ToolCalls,
			})
			for _, call := range action.ToolCalls {
				result := s.dispatchTool(ctx, call)
				s.logger.Debug("tool executed", "name", call.Name, "id", call.ID)
				*history = append(*history, models.Message{
					Role:       models.RoleTool,
					Content:    result,
					ToolCallID: call.ID,
				})
			}

		default:
			return fmt.Errorf("llm returned empty action")
		}
	}

	return fmt.Errorf("agent exceeded %d steps without final answer", maxSteps)
}

// readStream drains an LLM stream channel, forwarding text chunks to outputCh
// and returning the final AgentAction once the stream is complete.
func (s *SmartCalendarService) readStream(
	ctx context.Context,
	eventCh <-chan models.StreamEvent,
	outputCh chan<- models.Chunk,
) (models.AgentAction, error) {
	for event := range eventCh {
		switch {
		case event.Err != nil:
			return models.AgentAction{}, event.Err
		case event.Chunk != "":
			if !sendChunk(ctx, outputCh, event.Chunk) {
				return models.AgentAction{}, ctx.Err()
			}
		case event.Action != nil:
			return *event.Action, nil
		}
	}
	return models.AgentAction{}, fmt.Errorf("llm stream closed without action")
}

// dispatchTool executes a single tool call and returns its result as a JSON string
// suitable for feeding back into the LLM.
func (s *SmartCalendarService) dispatchTool(ctx context.Context, call models.ToolCall) string {
	switch call.Name {
	case "create_event":
		var args struct {
			CalendarID string       `json:"calendar_id"`
			Event      models.Event `json:"event"`
			Attendees  []string     `json:"attendees"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError(err)
		}
		if args.Attendees != nil {
			args.Event.Attendees = args.Attendees
		}
		ev, err := s.calendar.CreateEvent(ctx, args.CalendarID, args.Event)
		return toolResult(ev, err)

	case "delete_event":
		var args struct {
			CalendarID string `json:"calendar_id"`
			EventID    string `json:"event_id"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError(err)
		}
		err := s.calendar.DeleteEvent(ctx, args.CalendarID, args.EventID)
		return toolResult(map[string]string{"status": "deleted", "event_id": args.EventID}, err)

	case "get_event":
		var args struct {
			CalendarID string `json:"calendar_id"`
			EventID    string `json:"event_id"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError(err)
		}
		ev, err := s.calendar.GetEvent(ctx, args.CalendarID, args.EventID)
		return toolResult(ev, err)

	case "list_calendars":
		cals, err := s.calendar.ListCalendars(ctx)
		return toolResult(cals, err)

	case "list_events":
		var args struct {
			CalendarID string    `json:"calendar_id"`
			From       time.Time `json:"from"`
			To         time.Time `json:"to"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError(err)
		}
		events, err := s.calendar.ListEvents(ctx, args.CalendarID, args.From, args.To)
		return toolResult(events, err)

	case "respond_to_event":
		var args struct {
			CalendarID string               `json:"calendar_id"`
			EventID    string               `json:"event_id"`
			Response   models.EventResponse `json:"response"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError(err)
		}
		err := s.calendar.RespondToEvent(ctx, args.CalendarID, args.EventID, args.Response)
		return toolResult(map[string]string{"status": "ok"}, err)

	case "suggest_time":
		var req models.SuggestTimeRequest
		if err := json.Unmarshal([]byte(call.Arguments), &req); err != nil {
			return toolError(err)
		}
		slots, err := s.calendar.SuggestTime(ctx, req)
		return toolResult(slots, err)

	case "update_event":
		var args struct {
			CalendarID string       `json:"calendar_id"`
			Event      models.Event `json:"event"`
		}
		if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
			return toolError(err)
		}
		ev, err := s.calendar.UpdateEvent(ctx, args.CalendarID, args.Event)
		return toolResult(ev, err)

	default:
		return toolError(fmt.Errorf("unknown tool %q", call.Name))
	}
}

func toolResult(v any, err error) string {
	if err != nil {
		return toolError(err)
	}
	b, mErr := json.Marshal(v)
	if mErr != nil {
		return toolError(mErr)
	}
	return string(b)
}

func toolError(err error) string {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(b)
}

func sendChunk(ctx context.Context, ch chan<- models.Chunk, text string) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- models.Chunk{Text: text}:
		return true
	}
}

func sendDone(ctx context.Context, ch chan<- models.Chunk) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- models.Chunk{Done: true}:
		return true
	}
}

func recv(ctx context.Context, inputCh <-chan string) (string, bool) {
	select {
	case <-ctx.Done():
		return "", false
	case msg, ok := <-inputCh:
		return msg, ok
	}
}
