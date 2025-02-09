package dashboard

import (
	"errors"
	"image/png"
	"log/slog"
	"net/http"
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
	cfg     Config
	server  *http.Server
	Encoder *png.Encoder
}

func (s *Server) Start() {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server error", slog.Any("err", err))
		return
	}
}
