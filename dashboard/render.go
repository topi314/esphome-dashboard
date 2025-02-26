package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/sergeymakinen/go-bmp"

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
		"hasIndex":            hasIndex,
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
		"formatTimeToHour":    formatTimeToHour,
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

func (s *Server) renderDashboard(ctx context.Context, dashboard string, pageIndex int, width int, height int, format string) (io.Reader, int, string, error) {
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
		return nil, 0, "", fmt.Errorf("failed to run chromedp: %w", err)
	}

	return s.reencodeImage(bytes.NewReader(res), format)
}

func (s *Server) reencodeImage(r io.Reader, format string) (io.Reader, int, string, error) {
	decoded, err := png.Decode(r)
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to decode png: %w", err)
	}

	decoded.ColorModel().Convert(decoded.At(0, 0))

	paletted := image.NewPaletted(decoded.Bounds(), []color.Color{
		color.RGBA{R: 0, G: 0, B: 0, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})
	for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y; y++ {
		for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
			paletted.Set(x, y, decoded.At(x, y))
		}
	}

	encodedBuf := new(bytes.Buffer)
	var contentType string
	switch format {
	case "png":
		if err = s.pngEncoder.Encode(encodedBuf, paletted); err != nil {
			return nil, 0, "", fmt.Errorf("failed to encode png: %w", err)
		}
		contentType = "image/png"
	case "jpeg":
		if err = jpeg.Encode(encodedBuf, paletted, &jpeg.Options{Quality: 100}); err != nil {
			return nil, 0, "", fmt.Errorf("failed to encode jpeg: %w", err)
		}
		contentType = "image/jpeg"
	case "bmp":
		if err = bmp.Encode(encodedBuf, paletted); err != nil {
			return nil, 0, "", fmt.Errorf("failed to encode bmp: %w", err)
		}
		contentType = "image/bmp"
	default:
		return nil, 0, "", fmt.Errorf("unsupported format: %s", format)
	}

	return encodedBuf, encodedBuf.Len(), contentType, nil
}
