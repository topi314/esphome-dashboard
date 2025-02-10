package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func New(url string, token string) *Client {
	return &Client{
		url:   url,
		token: token,
		client: &http.Client{
			Timeout: 10,
		},
	}
}

type Client struct {
	url    string
	token  string
	client *http.Client
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.client.Do(req)
}

func (c *Client) GetState(ctx context.Context, entityID string) (State, error) {
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/states/"+entityID, nil)
	if err != nil {
		return State{}, fmt.Errorf("failed to create entity state request: %w", err)
	}

	rs, err := c.Do(rq)
	if err != nil {
		return State{}, fmt.Errorf("failed to get entity state: %w", err)
	}
	defer rs.Body.Close()

	var state State
	if err = json.NewDecoder(rs.Body).Decode(&state); err != nil {
		return State{}, fmt.Errorf("failed to decode entity state: %w", err)
	}

	return state, nil
}

func (c *Client) GetCalendar(ctx context.Context, entityID string) ([]CalendarEvent, error) {
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/calendars/"+entityID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar request: %w", err)
	}

	rs, err := c.Do(rq)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}
	defer rs.Body.Close()

	var events []CalendarEvent
	if err = json.NewDecoder(rs.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode calendar: %w", err)
	}

	return events, nil
}
