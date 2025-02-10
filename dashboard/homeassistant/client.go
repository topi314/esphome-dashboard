package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func New(url string, token string) *Client {
	return &Client{
		url:   url,
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Client struct {
	url    string
	token  string
	client *http.Client
}

func (c *Client) Do(rq *http.Request) (*http.Response, error) {
	rq.Header.Set("Authorization", "Bearer "+c.token)
	return c.client.Do(rq)
}

func (c *Client) Test(ctx context.Context) (string, error) {
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create test request: %w", err)
	}

	rs, err := c.Do(rq)
	if err != nil {
		return "", fmt.Errorf("failed to test connection: %w", err)
	}
	defer rs.Body.Close()

	var status Status
	if err = json.NewDecoder(rs.Body).Decode(&status); err != nil {
		return "", fmt.Errorf("failed to decode test response: %w", err)
	}

	return status.Message, nil
}

func (c *Client) GetState(ctx context.Context, entityID string) (EntityState, error) {
	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/states/"+entityID, nil)
	if err != nil {
		return EntityState{}, fmt.Errorf("failed to create entity state request: %w", err)
	}

	rs, err := c.Do(rq)
	if err != nil {
		return EntityState{}, fmt.Errorf("failed to get entity state: %w", err)
	}
	defer rs.Body.Close()

	if rs.StatusCode != http.StatusOK {
		return EntityState{}, fmt.Errorf("failed to get calendar: %s", rs.Status)
	}

	var state EntityState
	if err = json.NewDecoder(rs.Body).Decode(&state); err != nil {
		return EntityState{}, fmt.Errorf("failed to decode entity state: %w", err)
	}

	return state, nil
}

func (c *Client) GetCalendar(ctx context.Context, entityID string, start time.Time, end time.Time) ([]CalendarEvent, error) {
	v := url.Values{
		"start": {start.Format(time.RFC3339)},
		"end":   {end.Format(time.RFC3339)},
	}

	rq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/api/calendars/"+entityID+"?"+v.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar request: %w", err)
	}

	rs, err := c.Do(rq)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}
	defer rs.Body.Close()

	if rs.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get calendar: %s", rs.Status)
	}

	var events []CalendarEvent
	if err = json.NewDecoder(rs.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode calendar: %w", err)
	}

	return events, nil
}

func (c *Client) CallService(ctx context.Context, domain string, service string, serviceData io.Reader, returnResponse bool) (Response, error) {
	u := fmt.Sprintf("%s/api/services/%s/%s", c.url, domain, service)
	if returnResponse {
		u += "?return_response"
	}

	rq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, serviceData)
	if err != nil {
		return Response{}, fmt.Errorf("failed to create service call request: %w", err)
	}

	rq.Header.Set("Content-Type", "application/json")

	rs, err := c.Do(rq)
	if err != nil {
		return Response{}, fmt.Errorf("failed to call service: %w", err)
	}
	defer rs.Body.Close()

	if rs.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("failed to call service: %s", rs.Status)
	}

	var response Response
	if returnResponse {
		if err = json.NewDecoder(rs.Body).Decode(&response); err != nil {
			return Response{}, fmt.Errorf("failed to decode service response: %w", err)
		}
	} else {
		if err = json.NewDecoder(rs.Body).Decode(&response.ChangedStates); err != nil {
			return Response{}, fmt.Errorf("failed to decode service response: %w", err)
		}
	}

	return response, nil
}
