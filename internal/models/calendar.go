package models

import "time"

type EventResponse string

const (
	EventResponseAccepted  EventResponse = "accepted"
	EventResponseDeclined  EventResponse = "declined"
	EventResponseTentative EventResponse = "tentative"
)

type Event struct {
	ID          string
	CalendarID  string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	Attendees   []string
	Status      string
}

type Calendar struct {
	ID       string
	Summary  string
	TimeZone string
	Primary  bool
}

type TimeSlot struct {
	Start time.Time
	End   time.Time
}

type SuggestTimeRequest struct {
	Attendees      []string
	DurationMin    int
	EarliestStart  time.Time
	LatestEnd      time.Time
	PreferredStart time.Time
}
