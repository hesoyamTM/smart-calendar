package google

import (
	"time"

	"github.com/hesoyamTM/smart-calendar/internal/models"
	gcalendar "google.golang.org/api/calendar/v3"
)

func toGoogleEvent(e models.Event) *gcalendar.Event {
	ge := &gcalendar.Event{
		Id:          e.ID,
		Summary:     e.Summary,
		Description: e.Description,
		Location:    e.Location,
		Start:       &gcalendar.EventDateTime{DateTime: e.Start.Format(time.RFC3339)},
		End:         &gcalendar.EventDateTime{DateTime: e.End.Format(time.RFC3339)},
	}
	for _, a := range e.Attendees {
		ge.Attendees = append(ge.Attendees, &gcalendar.EventAttendee{Email: a})
	}
	return ge
}

func fromGoogleEvent(ge *gcalendar.Event) models.Event {
	e := models.Event{
		ID:          ge.Id,
		Summary:     ge.Summary,
		Description: ge.Description,
		Location:    ge.Location,
		Status:      ge.Status,
	}
	if ge.Start != nil {
		e.Start, _ = parseGoogleTime(ge.Start)
	}
	if ge.End != nil {
		e.End, _ = parseGoogleTime(ge.End)
	}
	for _, a := range ge.Attendees {
		e.Attendees = append(e.Attendees, a.Email)
	}
	return e
}

func parseGoogleTime(edt *gcalendar.EventDateTime) (time.Time, error) {
	if edt.DateTime != "" {
		return time.Parse(time.RFC3339, edt.DateTime)
	}
	return time.Parse("2006-01-02", edt.Date)
}
