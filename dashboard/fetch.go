package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/topi314/esphome-dashboard/dashboard/homeassistant"
)

func (s *Server) fetchHomeAssistantData(ctx context.Context, config DashboardHomeAssistantConfig) HomeAssistantRenderData {
	homeAssistantRenderData := HomeAssistantRenderData{
		Entities:  make(map[string]homeassistant.EntityState),
		Calendars: make(map[string][]homeassistant.CalendarEvent),
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

	for _, calendar := range config.Calendars {
		events, err := s.homeAssistant.GetCalendar(ctx, calendar.ID, start, start.AddDate(0, 0, calendar.Days))
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.ErrorContext(ctx, "failed to get calendar", slog.String("calendar", calendar.Name), slog.String("entity_id", calendar.ID), slog.Any("err", err))
			continue
		}
		homeAssistantRenderData.Calendars[calendar.Name] = events
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
