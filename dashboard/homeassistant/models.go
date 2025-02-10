package homeassistant

import (
	"fmt"
	"time"
)

type State struct {
	Attributes  map[string]any `json:"attributes"`
	EntityID    string         `json:"entity_id"`
	LastChanged time.Time      `json:"last_changed"`
	LastUpdated time.Time      `json:"last_updated"`
	State       string         `json:"state"`
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
