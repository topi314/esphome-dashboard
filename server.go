package main

import (
	"errors"
	"log/slog"
	"net/http"
)

func newServer(baseDir string, addr string, r http.Handler) *server {
	return &server{
		BaseDir: baseDir,
		Server: &http.Server{
			Addr:    addr,
			Handler: r,
		},
	}
}

type server struct {
	BaseDir string
	Server  *http.Server
}

func (s *server) Start() {
	go func() {
		if err := s.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", slog.Any("err", err))
			return
		}
	}()
}
