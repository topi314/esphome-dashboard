package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"image/png"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"

	"github.com/topi314/esphome-dashboard/dashboard/homeassistant"
)

type RenderData struct {
	PageIndex     int
	PageCount     int
	Pages         []PageRenderData
	Vars          map[string]any
	HomeAssistant HomeAssistantRenderData
}

func (r RenderData) Page() PageRenderData {
	return r.Pages[r.PageIndex]
}

type PageRenderData struct {
	Index int
	Vars  map[string]any
}

type HomeAssistantRenderData struct {
	Entities  map[string]homeassistant.EntityState
	Calendars map[string][]CalendarDay
	Services  map[string]homeassistant.Response
}

type CalendarDay struct {
	Time    time.Time
	IsPast  bool
	IsToday bool
	Events  []homeassistant.CalendarEvent
}

func (s *Server) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"seq":                 seq,
		"now":                 time.Now,
		"dict":                dict,
		"reverse":             reverse,
		"parseTime":           parseTime,
		"convertNewLinesToBR": convertNewLinesToBR,
		"safeHTML":            safeHTML,
		"safeCSS":             safeCSS,
		"safeHTMLAttr":        safeHTMLAttr,
		"safeURL":             safeURL,
		"safeJS":              safeJS,
		"safeJSStr":           safeJSStr,
		"safeSrcset":          safeSrcset,
		"formatTimeToDay":     formatTimeToDay,
		"formatTimeToRelDay":  formatTimeToRelDay,
	}
}

func (s *Server) executeDashboard(ctx context.Context, base Base) (io.Reader, int, error) {
	baseTemplate, err := template.New("base").
		Funcs(s.templateFuncs()).
		Parse(string(base.Body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse base template: %w", err)
	}

	var pageRenderData []PageRenderData
	for _, p := range base.Pages {
		_, err = baseTemplate.New(strings.TrimSuffix(filepath.Base(p.Name), filepath.Ext(p.Name))).
			Funcs(s.templateFuncs()).
			Parse(string(p.Body))
		if err != nil {
			return nil, 0, fmt.Errorf("failed to parse page template: %w", err)
		}

		pageRenderData = append(pageRenderData, PageRenderData{
			Index: p.Index,
			Vars:  p.Vars,
		})
	}

	if _, err = baseTemplate.New("page").
		Funcs(s.templateFuncs()).
		Parse(string(base.Pages[base.PageIndex].Body)); err != nil {
		return nil, 0, fmt.Errorf("failed to parse page template: %w", err)
	}

	if _, err = baseTemplate.ParseFS(s.templates, "templates/*.gohtml"); err != nil {
		return nil, 0, fmt.Errorf("failed to parse templates: %w", err)
	}

	slog.DebugContext(ctx, "loaded templates", slog.String("templates", baseTemplate.DefinedTemplates()))

	homeAssistantRenderData := s.fetchHomeAssistantData(ctx, base.Config.HomeAssistant)

	data := RenderData{
		PageIndex:     base.PageIndex,
		PageCount:     len(base.Config.Pages),
		Pages:         pageRenderData,
		Vars:          base.Vars,
		HomeAssistant: homeAssistantRenderData,
	}

	var buf bytes.Buffer
	if err = baseTemplate.ExecuteTemplate(&buf, "base", data); err != nil {
		return nil, 0, fmt.Errorf("failed to execute template: %w", err)
	}

	return &buf, buf.Len(), nil
}

func (s *Server) renderDashboard(ctx context.Context, dashboard string, pageIndex int, width int, height int) (io.Reader, int, error) {
	var cancel context.CancelFunc
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var res []byte
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(width), int64(height)),
		chromedp.Navigate(fmt.Sprintf("http://localhost:%d/dashboards/%s/pages/%d?html=1", s.cfg.ListenPort, dashboard, pageIndex)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			res, err = page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithFromSurface(true).
				WithOptimizeForSpeed(true).
				Do(ctx)
			return err
		}),
	); err != nil {
		return nil, 0, fmt.Errorf("failed to run chromedp: %w", err)
	}

	return s.reencodePNG(bytes.NewReader(res))
}

func (s *Server) reencodePNG(r io.Reader) (io.Reader, int, error) {
	decoded, err := png.Decode(r)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode png: %w", err)
	}

	encodedBuf := new(bytes.Buffer)
	if err = s.encoder.Encode(encodedBuf, decoded); err != nil {
		return nil, 0, fmt.Errorf("failed to encode png: %w", err)
	}

	return encodedBuf, encodedBuf.Len(), nil
}
