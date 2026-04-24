// Package claude provides an LLM client backed by the Claude API.
package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/hesoyamTM/smart-calendar/internal/models"
)

type Client struct {
	api anthropic.Client
}

// New creates a Client authenticated with the given API key.
func New(cfg Config) *Client {
	return &Client{
		api: anthropic.NewClient(
			option.WithAPIKey(cfg.APIKey),
			option.WithBaseURL(cfg.BaseURL),
		),
	}
}

type blockAccum struct {
	typ  string
	id   string
	name string
	buf  strings.Builder
}

// NextAction streams the Claude response into the returned channel.
// Text deltas are sent as StreamEvent{Chunk: "..."}.
// When complete, a single StreamEvent{Action: &action} is sent and the channel is closed.
// On error, StreamEvent{Err: err} is sent and the channel is closed.
func (c *Client) NextAction(ctx context.Context, history []models.Message) <-chan models.StreamEvent {
	ch := make(chan models.StreamEvent, 32)

	go func() {
		defer close(ch)

		system, messages := convertHistory(history)
		adaptive := anthropic.ThinkingConfigAdaptiveParam{}
		stream := c.api.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
			Model:     anthropic.ModelClaudeOpus4_7,
			MaxTokens: 16000,
			System:    []anthropic.TextBlockParam{{Text: system}},
			Messages:  messages,
			Tools:     calendarTools(),
			Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		})
		defer stream.Close()

		var blocks []blockAccum

		for stream.Next() {
			switch e := stream.Current().AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				idx := int(e.Index)
				for len(blocks) <= idx {
					blocks = append(blocks, blockAccum{})
				}
				blocks[idx].typ = e.ContentBlock.Type
				blocks[idx].id = e.ContentBlock.ID
				blocks[idx].name = e.ContentBlock.Name

			case anthropic.ContentBlockDeltaEvent:
				idx := int(e.Index)
				if idx >= len(blocks) {
					continue
				}
				switch d := e.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					blocks[idx].buf.WriteString(d.Text)
					if blocks[idx].typ == "text" {
						select {
						case ch <- models.StreamEvent{Chunk: d.Text}:
						case <-ctx.Done():
							return
						}
					}
				case anthropic.InputJSONDelta:
					blocks[idx].buf.WriteString(d.PartialJSON)
				}
			}
		}
		if err := stream.Err(); err != nil {
			ch <- models.StreamEvent{Err: fmt.Errorf("claude stream: %w", err)}
			return
		}

		action, err := buildAction(blocks)
		if err != nil {
			ch <- models.StreamEvent{Err: err}
			return
		}
		ch <- models.StreamEvent{Action: &action}
	}()

	return ch
}

// convertHistory converts the given history into a system prompt and a list of messages.
func convertHistory(history []models.Message) (string, []anthropic.MessageParam) {
	system := ""
	messages := make([]anthropic.MessageParam, 0, len(history))

	i := 0
	for i < len(history) {
		msg := history[i]
		switch msg.Role {
		case models.RoleSystem:
			system = msg.Content
			i++

		case models.RoleUser:
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
			i++

		case models.RoleAssistant:
			if len(msg.ToolCalls) > 0 {
				blocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolCalls)+1)
				if msg.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(msg.Content))
				}
				for _, tc := range msg.ToolCalls {
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, json.RawMessage(tc.Arguments), tc.Name))
				}
				messages = append(messages, anthropic.NewAssistantMessage(blocks...))
			} else {
				messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
			}
			i++

		case models.RoleTool:
			var results []anthropic.ContentBlockParamUnion
			for i < len(history) && history[i].Role == models.RoleTool {
				results = append(results, anthropic.NewToolResultBlock(history[i].ToolCallID, history[i].Content, false))
				i++
			}
			messages = append(messages, anthropic.NewUserMessage(results...))
		}
	}
	return system, messages
}

// buildAction assembles an AgentAction from the accumulated stream blocks.
func buildAction(blocks []blockAccum) (models.AgentAction, error) {
	var action models.AgentAction

	for _, b := range blocks {
		switch b.typ {
		case "text":
			action.FinalAnswer = b.buf.String()

		case "tool_use":
			if b.name == "ask_clarification" {
				var args struct {
					Question string `json:"question"`
				}
				if err := json.Unmarshal([]byte(b.buf.String()), &args); err != nil {
					return models.AgentAction{}, fmt.Errorf("parse clarification args: %w", err)
				}
				return models.AgentAction{ClarifyingQuestion: args.Question}, nil
			}
			action.ToolCalls = append(action.ToolCalls, models.ToolCall{
				ID:        b.id,
				Name:      b.name,
				Arguments: b.buf.String(),
			})
		}
	}

	if len(action.ToolCalls) > 0 {
		action.FinalAnswer = ""
	}

	return action, nil
}
