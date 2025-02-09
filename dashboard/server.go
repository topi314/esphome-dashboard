package dashboard

import (
	"context"
	"errors"
	"image/png"
	"log/slog"
	"net"
	"net/http"

	"github.com/chromedp/chromedp"
)

func New(cfg Config) *Server {
	s := &Server{
		cfg: cfg,
		Encoder: &png.Encoder{
			CompressionLevel: png.BestCompression,
		},
	}

	s.server = &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: s.Routes(),
	}

	return s
}

type Server struct {
	cfg       Config
	server    *http.Server
	Encoder   *png.Encoder
	chromeCtx context.Context
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

	s.chromeCtx = chromeCtx

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", slog.Any("err", err))
		return
	}
}
