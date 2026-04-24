package claude

import "github.com/anthropics/anthropic-sdk-go"

func calendarTools() []anthropic.ToolUnionParam {
	tool := func(name, desc string, props map[string]any, required []string) anthropic.ToolUnionParam {
		return anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
			Name:        name,
			Description: anthropic.String(desc),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: props,
				Required:   required,
			},
		}}
	}

	eventProps := map[string]any{
		"summary":     map[string]any{"type": "string", "description": "Event title"},
		"description": map[string]any{"type": "string", "description": "Event description"},
		"location":    map[string]any{"type": "string", "description": "Event location"},
		"start":       map[string]any{"type": "string", "description": "Start time in RFC3339 format"},
		"end":         map[string]any{"type": "string", "description": "End time in RFC3339 format"},
		"attendees":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Attendee email addresses"},
	}

	return []anthropic.ToolUnionParam{
		tool("ask_clarification",
			"Ask the user a clarifying question when information is missing or ambiguous.",
			map[string]any{
				"question": map[string]any{"type": "string", "description": "The question to ask the user"},
			},
			[]string{"question"},
		),
		tool("create_event",
			"Create a new event in Google Calendar.",
			map[string]any{
				"calendar_id": map[string]any{"type": "string", "description": "Calendar ID ('primary' for the main calendar)"},
				"event":       map[string]any{"type": "object", "properties": eventProps, "required": []string{"summary", "start", "end"}},
			},
			[]string{"calendar_id", "event"},
		),
		tool("delete_event",
			"Delete an event from Google Calendar.",
			map[string]any{
				"calendar_id": map[string]any{"type": "string", "description": "Calendar ID"},
				"event_id":    map[string]any{"type": "string", "description": "Event ID to delete"},
			},
			[]string{"calendar_id", "event_id"},
		),
		tool("get_event",
			"Get details of a specific event.",
			map[string]any{
				"calendar_id": map[string]any{"type": "string", "description": "Calendar ID"},
				"event_id":    map[string]any{"type": "string", "description": "Event ID"},
			},
			[]string{"calendar_id", "event_id"},
		),
		tool("list_calendars",
			"List all calendars available to the user.",
			map[string]any{},
			nil,
		),
		tool("list_events",
			"List events in a calendar within a time range.",
			map[string]any{
				"calendar_id": map[string]any{"type": "string", "description": "Calendar ID"},
				"from":        map[string]any{"type": "string", "description": "Start of range in RFC3339 format"},
				"to":          map[string]any{"type": "string", "description": "End of range in RFC3339 format"},
			},
			[]string{"calendar_id", "from", "to"},
		),
		tool("respond_to_event",
			"Accept, decline, or tentatively accept a calendar invitation.",
			map[string]any{
				"calendar_id": map[string]any{"type": "string", "description": "Calendar ID"},
				"event_id":    map[string]any{"type": "string", "description": "Event ID"},
				"response":    map[string]any{"type": "string", "enum": []string{"accepted", "declined", "tentative"}},
			},
			[]string{"calendar_id", "event_id", "response"},
		),
		tool("suggest_time",
			"Suggest available meeting time slots for a group of attendees.",
			map[string]any{
				"attendees":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Attendee email addresses"},
				"duration_min":    map[string]any{"type": "integer", "description": "Meeting duration in minutes"},
				"earliest_start":  map[string]any{"type": "string", "description": "Earliest acceptable start time in RFC3339 format"},
				"latest_end":      map[string]any{"type": "string", "description": "Latest acceptable end time in RFC3339 format"},
				"preferred_start": map[string]any{"type": "string", "description": "Preferred start time in RFC3339 format (optional)"},
			},
			[]string{"attendees", "duration_min", "earliest_start", "latest_end"},
		),
		tool("update_event",
			"Update an existing event in Google Calendar.",
			map[string]any{
				"calendar_id": map[string]any{"type": "string", "description": "Calendar ID"},
				"event": map[string]any{
					"type": "object",
					"properties": func() map[string]any {
						p := map[string]any{"id": map[string]any{"type": "string", "description": "Event ID to update"}}
						for k, v := range eventProps {
							p[k] = v
						}
						return p
					}(),
					"required": []string{"id"},
				},
			},
			[]string{"calendar_id", "event"},
		),
	}
}
