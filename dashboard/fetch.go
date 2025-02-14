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
	entities, err := s.fetchHomeAssistantEntities(ctx, config.Entities)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch home assistant entities", slog.Any("err", err))
	}
	calendars, err := s.fetchHomeAssistantCalendars(ctx, config.Calendars)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch home assistant calendars", slog.Any("err", err))
	}
	services, err := s.fetchHomeAssistantServices(ctx, config.Services)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch home assistant services", slog.Any("err", err))
	}

	return HomeAssistantRenderData{
		Entities:  entities,
		Calendars: calendars,
		Services:  services,
	}
}

func (s *Server) fetchHomeAssistantEntities(ctx context.Context, entities []EntityConfig) (map[string]homeassistant.EntityState, error) {
	states := make(map[string]homeassistant.EntityState)
	for _, entity := range entities {
		state, err := s.homeAssistant.GetState(ctx, entity.ID)
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to get entity state", slog.String("entity", entity.Name), slog.String("entity_id", entity.ID), slog.Any("err", err))
			continue
		}
		states[entity.Name] = state
	}

	return states, nil
}

func (s *Server) fetchHomeAssistantCalendars(ctx context.Context, calendars []CalendarConfig) (map[string][]CalendarDay, error) {
	year, month, day := time.Now().Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	start = start.AddDate(0, 0, -weekdayToIndex(start.Weekday())) // move start at the beginning of the week

	days := make(map[string][]CalendarDay)
	for _, calendar := range calendars {
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

		days[calendar.Name] = fillAndSortCalendarDays(calendar, allEvents, start)
	}

	return days, nil
}

func fillAndSortCalendarDays(calendar CalendarConfig, events []homeassistant.CalendarEvent, start time.Time) []CalendarDay {
	nowYear, nowMonth, nowDay := time.Now().Date()
	now := time.Date(nowYear, nowMonth, nowDay, 0, 0, 0, 0, time.UTC)

	end := start.AddDate(0, 0, 28) // add 4 weeks

	// fill in all 28 days
	days := make([]CalendarDay, 0, 28)
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		days = append(days, CalendarDay{
			Time:    d,
			IsPast:  d.Before(now),
			IsToday: d.Equal(now),
			Events:  nil,
		})
	}

	for _, event := range events {
		startDay := event.Start.Day()
		endDay := event.End.Day()

		// find the index of the start day
		firstDayIndex := slices.IndexFunc(days, func(cDay CalendarDay) bool {
			return startDay.Equal(cDay.Time)
		})
		// if the start day is not in the range, skip the event
		if firstDayIndex == -1 {
			continue
		}

		if startDay.Equal(endDay) {
			// add the event to the start day
			days[firstDayIndex].Events = append(days[firstDayIndex].Events, event)
			continue
		}

		// add the event to all days between start and end
		for i := firstDayIndex; i < len(days); i++ {
			if days[i].Time.Equal(endDay) {
				break
			}
			days[i].Events = append(days[i].Events, event)
		}
	}

	for i := range days {
		slices.SortFunc(days[i].Events, func(a, b homeassistant.CalendarEvent) int {
			if a.Start.DateTime.Before(b.Start.DateTime) {
				return -1
			} else if a.Start.DateTime.After(b.Start.DateTime) {
				return 1
			} else {
				return 0
			}
		})
	}

	if calendar.SkipPastEvents {
		for i := range days {
			if days[i].IsPast {
				days[i].Events = nil
			}
		}
	}

	if calendar.MaxEvents > 0 {
		var totalEvents int
		for i := range days {
			if totalEvents > calendar.MaxEvents {
				days[i].Events = nil
				continue
			}
			totalEvents += len(days[i].Events)
			if totalEvents > calendar.MaxEvents {
				days[i].Events = days[i].Events[:1+len(days[i].Events)-(totalEvents-calendar.MaxEvents)]
			}
		}
	}

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

func (s *Server) fetchHomeAssistantServices(ctx context.Context, services []ServiceConfig) (map[string]homeassistant.Response, error) {
	responses := make(map[string]homeassistant.Response)
	for _, service := range services {
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
		responses[service.Name] = response
	}

	return responses, nil
}
