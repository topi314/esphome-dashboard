package homeassistant

import (
	"fmt"
	"time"
)

type Status struct {
	Message string `json:"message"`
}

type EntityState struct {
	EntityID    string         `json:"entity_id"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
	State       string         `json:"state"`
	Attributes  map[string]any `json:"attributes"`
}

type CalendarEvent struct {
	Summary     string `json:"summary"`
	Start       Date   `json:"start"`
	End         Date   `json:"end"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

func (e CalendarEvent) StartDay() time.Time {
	year, month, day := e.Start.Time().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func (e CalendarEvent) EndDay() time.Time {
	end := e.End.Time()
	year, month, day := end.Date()
	// if the event ends at 00:00 then it is still part of the previous day
	if end.Hour() == 0 && end.Minute() == 0 && end.Second() == 0 {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	}
	return end
}

func (e CalendarEvent) IsFullDay(day time.Time) bool {
	startTime := e.Start.Time()
	endTime := e.End.Time()

	// if the event is a full day event (starts at 00:00)
	if startTime.Equal(day) {
		return true
	}

	// if the event is over multiple days the events between the start and end day are full day events
	if startTime.Before(day) && endTime.After(day) {
		return true
	}

	return false
}

type Date struct {
	DateTime time.Time `json:"dateTime"`
	Date     string    `json:"date"`
}

func (d Date) Time() time.Time {
	if d.DateTime.IsZero() {
		date, err := time.Parse(time.DateOnly, d.Date)
		if err != nil {
			panic(fmt.Errorf("failed to parse date: %w", err))
		}
		return date
	}
	return d.DateTime
}

type Response struct {
	ChangedStates   []EntityState  `json:"changed_states"`
	ServiceResponse map[string]any `json:"service_response"`
}
