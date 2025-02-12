package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/topi314/esphome-dashboard/dashboard/homeassistant"
)

func (s *Server) fetchHomeAssistantData(ctx context.Context, config DashboardHomeAssistantConfig) HomeAssistantRenderData {
	homeAssistantRenderData := HomeAssistantRenderData{
		Entities:  make(map[string]homeassistant.EntityState),
		Calendars: make(map[string][]CalendarDay),
		Services:  make(map[string]homeassistant.Response),
	}
	for _, entity := range config.Entities {
		state, err := s.homeAssistant.GetState(ctx, entity.ID)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to get entity state", slog.String("entity", entity.Name), slog.String("entity_id", entity.ID), slog.Any("err", err))
			continue
		}
		homeAssistantRenderData.Entities[entity.Name] = state
	}

	year, month, day := time.Now().Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	start = start.AddDate(0, 0, -weekdayToIndex(start.Weekday())) // move start at the beginning of the week

	for _, calendar := range config.Calendars {
		end := start.AddDate(0, 0, calendar.Days)

		var allEvents []homeassistant.CalendarEvent
		for _, id := range calendar.IDs {
			events, err := s.homeAssistant.GetCalendar(ctx, id, start, end)
			if err != nil && !errors.Is(err, context.Canceled) {
				slog.ErrorContext(ctx, "failed to get calendar", slog.String("calendar", calendar.Name), slog.String("entity_id", id), slog.Any("err", err))
				continue
			}
			allEvents = append(allEvents, events...)
		}

		homeAssistantRenderData.Calendars[calendar.Name] = toCalendarDays(allEvents, start)
	}

	for _, service := range config.Services {
		data, err := json.Marshal(service.Data)
		if err != nil {
			slog.ErrorContext(ctx, "failed to marshal service data", slog.String("domain", service.Domain), slog.String("service", service.Service), slog.Any("err", err))
			continue
		}

		response, err := s.homeAssistant.CallService(ctx, service.Domain, service.Service, bytes.NewReader(data), service.ReturnResponse)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to call service", slog.String("domain", service.Domain), slog.String("service", service.Service), slog.Any("err", err))
			continue
		}
		homeAssistantRenderData.Services[service.Name] = response
	}

	return homeAssistantRenderData
}

func toCalendarDays(events []homeassistant.CalendarEvent, start time.Time) []CalendarDay {
	var days []CalendarDay
	for _, event := range events {
		year, month, day := event.Start.Time().Date()

		i := slices.IndexFunc(days, func(cDay CalendarDay) bool {
			dayYear, dayMonth, dayDay := cDay.Time.Date()
			return year == dayYear && month == dayMonth && day == dayDay
		})

		if i == -1 {
			days = append(days, CalendarDay{
				Time:   time.Date(year, month, day, 0, 0, 0, 0, time.Local),
				Events: []homeassistant.CalendarEvent{event},
			})
			continue
		}

		days[i].Events = append(days[i].Events, event)
	}

	year, month, day := time.Now().Date()
	now := time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 28) // add 4 weeks

	// fill in missing days with no events
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		i := slices.IndexFunc(days, func(cDay CalendarDay) bool {
			return d.Equal(cDay.Time)
		})
		if i == -1 {
			days = append(days, CalendarDay{
				Time:   d,
				Past:   d.Before(now),
				Events: []homeassistant.CalendarEvent{},
			})
		} else {
			slices.SortFunc(days[i].Events, func(a homeassistant.CalendarEvent, b homeassistant.CalendarEvent) int {
				if a.Start.Time().Before(b.Start.Time()) {
					return -1
				} else if a.Start.Time().After(b.Start.Time()) {
					return 1
				}
				return 0
			})
		}
	}

	slices.SortFunc(days, func(a CalendarDay, b CalendarDay) int {
		if a.Time.Before(b.Time) {
			return -1
		} else if a.Time.After(b.Time) {
			return 1
		}
		return 0
	})

	return days
}

func weekdayToIndex(weekday time.Weekday) int {
	switch weekday {
	case time.Monday:
		return 0
	case time.Tuesday:
		return 1
	case time.Wednesday:
		return 2
	case time.Thursday:
		return 3
	case time.Friday:
		return 4
	case time.Saturday:
		return 5
	case time.Sunday:
		return 6
	default:
		panic("invalid weekday")
	}
}
