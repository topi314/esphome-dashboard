package dashboard

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"image/png"
	"io"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type RenderOptions struct {
	Width     int
	Height    int
	PrintHTML bool
}

type RenderData struct {
	PageIndex int
	PageCount int
	Pages     []PageRenderData
	Vars      map[string]any
}

func (r RenderData) Page() PageRenderData {
	return r.Pages[r.PageIndex]
}

type PageRenderData struct {
	Index int
	Vars  map[string]any
}

func (s *Server) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"seq": seq,
	}
}

func (s *Server) renderDashboard(ctx context.Context, base Base, options RenderOptions) (io.Reader, int, error) {
	baseTemplate, err := template.New("base").
		Funcs(s.templateFuncs()).
		Parse(string(base.Body))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse base template: %w", err)
	}

	var pageRenderData []PageRenderData
	for _, p := range base.Pages {
		_, err = baseTemplate.New(p.Name).
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

	var buf bytes.Buffer
	if err = baseTemplate.ExecuteTemplate(&buf, "base", RenderData{
		PageIndex: base.PageIndex,
		PageCount: len(base.Config.Pages),
		Pages:     pageRenderData,
		Vars:      base.Vars,
	}); err != nil {
		return nil, 0, fmt.Errorf("failed to execute template: %w", err)
	}

	if options.PrintHTML {
		return &buf, buf.Len(), nil
	}

	var cancel context.CancelFunc
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	var res []byte
	if err = chromedp.Run(ctx,
		chromedp.EmulateViewport(int64(options.Width), int64(options.Height)),
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, buf.String()).Do(ctx)
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
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
	if err = s.Encoder.Encode(encodedBuf, decoded); err != nil {
		return nil, 0, fmt.Errorf("failed to encode png: %w", err)
	}

	return encodedBuf, encodedBuf.Len(), nil
}
