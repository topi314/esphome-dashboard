package dashboard

import (
	"context"
	"errors"
	"image/png"
	"io/fs"
	"log/slog"
	"net"
	"net/http"

	"github.com/chromedp/chromedp"

	"github.com/topi314/esphome-dashboard/dashboard/homeassistant"
)

func New(cfg Config, templates fs.FS) *Server {
	s := &Server{
		cfg:       cfg,
		templates: templates,
		encoder: &png.Encoder{
			CompressionLevel: png.BestCompression,
		},
		homeAssistant: homeassistant.New(cfg.HomeAssistant.URL(), cfg.HomeAssistant.Token),
	}

	s.server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: s.Routes(),
	}

	return s
}

type Server struct {
	cfg           Config
	templates     fs.FS
	server        *http.Server
	encoder       *png.Encoder
	homeAssistant *homeassistant.Client
}

func (s *Server) Start() {
	chromeCtx, chromeCancel := chromedp.NewContext(context.Background())
	defer chromeCancel()
	if err := chromedp.Run(chromeCtx, chromedp.Navigate("about:blank")); err != nil {
		slog.Error("failed to start chrome", slog.Any("err", err))
		return
	}

	s.server.BaseContext = func(listener net.Listener) context.Context {
		return chromeCtx
	}

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", slog.Any("err", err))
		return
	}
}
