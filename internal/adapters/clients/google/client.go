// Package google provides a client for the Google Calendar API.
package google

import (
	"context"
	"fmt"
	"time"

	"github.com/hesoyamTM/smart-calendar/internal/models"
	"golang.org/x/oauth2"
	gcalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Client struct {
	svc *gcalendar.Service
}

// New creates a Client using a token obtained externally.
func New(ctx context.Context, cfg Config, tok *oauth2.Token) (*Client, error) {
	httpClient := newOAuthConfig(cfg).Client(ctx, tok)

	svc, err := gcalendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("google calendar service: %w", err)
	}
	return &Client{svc: svc}, nil
}

func (c *Client) CreateEvent(ctx context.Context, calendarID string, e models.Event) (models.Event, error) {
	ge := toGoogleEvent(e)
	created, err := c.svc.Events.Insert(calendarID, ge).Context(ctx).Do()
	if err != nil {
		return models.Event{}, fmt.Errorf("create event: %w", err)
	}
	return fromGoogleEvent(created), nil
}

func (c *Client) DeleteEvent(ctx context.Context, calendarID, eventID string) error {
	if err := c.svc.Events.Delete(calendarID, eventID).Context(ctx).Do(); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}
	return nil
}

func (c *Client) GetEvent(ctx context.Context, calendarID, eventID string) (models.Event, error) {
	ge, err := c.svc.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return models.Event{}, fmt.Errorf("get event: %w", err)
	}
	return fromGoogleEvent(ge), nil
}

func (c *Client) ListCalendars(ctx context.Context) ([]models.Calendar, error) {
	var cals []models.Calendar
	err := c.svc.CalendarList.List().Context(ctx).Pages(ctx, func(page *gcalendar.CalendarList) error {
		for _, item := range page.Items {
			cals = append(cals, models.Calendar{
				ID:       item.Id,
				Summary:  item.Summary,
				TimeZone: item.TimeZone,
				Primary:  item.Primary,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list calendars: %w", err)
	}
	return cals, nil
}

func (c *Client) ListEvents(ctx context.Context, calendarID string, from, to time.Time) ([]models.Event, error) {
	var events []models.Event
	err := c.svc.Events.List(calendarID).
		TimeMin(from.Format(time.RFC3339)).
		TimeMax(to.Format(time.RFC3339)).
		SingleEvents(true).
		OrderBy("startTime").
		Context(ctx).
		Pages(ctx, func(page *gcalendar.Events) error {
			for _, ge := range page.Items {
				events = append(events, fromGoogleEvent(ge))
			}
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}

func (c *Client) RespondToEvent(ctx context.Context, calendarID, eventID string, resp models.EventResponse) error {
	ge, err := c.svc.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("respond to event (get): %w", err)
	}

	cal, err := c.svc.Calendars.Get(calendarID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("respond to event (get calendar): %w", err)
	}

	for _, a := range ge.Attendees {
		if a.Email == cal.Id {
			a.ResponseStatus = string(resp)
		}
	}

	if _, err := c.svc.Events.Update(calendarID, eventID, ge).Context(ctx).Do(); err != nil {
		return fmt.Errorf("respond to event (update): %w", err)
	}
	return nil
}

func (c *Client) SuggestTime(ctx context.Context, req models.SuggestTimeRequest) ([]models.TimeSlot, error) {
	items := make([]*gcalendar.FreeBusyRequestItem, len(req.Attendees))
	for i, a := range req.Attendees {
		items[i] = &gcalendar.FreeBusyRequestItem{Id: a}
	}

	fb, err := c.svc.Freebusy.Query(&gcalendar.FreeBusyRequest{
		TimeMin: req.EarliestStart.Format(time.RFC3339),
		TimeMax: req.LatestEnd.Format(time.RFC3339),
		Items:   items,
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("suggest time (freebusy): %w", err)
	}

	type interval struct{ start, end time.Time }
	var busy []interval
	for _, cal := range fb.Calendars {
		for _, b := range cal.Busy {
			s, _ := time.Parse(time.RFC3339, b.Start)
			e, _ := time.Parse(time.RFC3339, b.End)
			busy = append(busy, interval{s, e})
		}
	}

	duration := time.Duration(req.DurationMin) * time.Minute
	cursor := req.EarliestStart
	if !req.PreferredStart.IsZero() && req.PreferredStart.After(cursor) {
		cursor = req.PreferredStart
	}

	var slots []models.TimeSlot
	for cursor.Add(duration).Before(req.LatestEnd) || cursor.Add(duration).Equal(req.LatestEnd) {
		end := cursor.Add(duration)
		free := true
		for _, b := range busy {
			if cursor.Before(b.end) && end.After(b.start) {
				free = false
				cursor = b.end
				break
			}
		}
		if free {
			slots = append(slots, models.TimeSlot{Start: cursor, End: end})
			cursor = end
		}
	}

	return slots, nil
}

func (c *Client) UpdateEvent(ctx context.Context, calendarID string, e models.Event) (models.Event, error) {
	ge := toGoogleEvent(e)
	updated, err := c.svc.Events.Update(calendarID, e.ID, ge).Context(ctx).Do()
	if err != nil {
		return models.Event{}, fmt.Errorf("update event: %w", err)
	}
	return fromGoogleEvent(updated), nil
}
