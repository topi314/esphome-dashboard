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
