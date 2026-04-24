package service

import (
	"context"
	"time"

	"github.com/hesoyamTM/smart-calendar/internal/models"
)

type CalendarClient interface {
	CreateEvent(ctx context.Context, calendarID string, e models.Event) (models.Event, error)
	DeleteEvent(ctx context.Context, calendarID, eventID string) error
	GetEvent(ctx context.Context, calendarID, eventID string) (models.Event, error)
	ListCalendars(ctx context.Context) ([]models.Calendar, error)
	ListEvents(ctx context.Context, calendarID string, from, to time.Time) ([]models.Event, error)
	RespondToEvent(ctx context.Context, calendarID, eventID string, resp models.EventResponse) error
	SuggestTime(ctx context.Context, req models.SuggestTimeRequest) ([]models.TimeSlot, error)
	UpdateEvent(ctx context.Context, calendarID string, e models.Event) (models.Event, error)
}

type LLMClient interface {
	NextAction(ctx context.Context, history []models.Message) <-chan models.StreamEvent
}
