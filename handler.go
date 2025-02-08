package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

type Action string

const (
	ActionRefresh   Action = "refresh"
	ActionNextPage  Action = "next_page"
	ActionLastPage  Action = "last_page"
	ActionPrevPage  Action = "prev_page"
	ActionFirstPage Action = "first_page"
)

func (s *server) getControl(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "getControl")
	dashboard := r.PathValue("dashboard")

	query := r.URL.Query()
	action := Action(query.Get("action"))
	lastPageStr := query.Get("page")
	lastPage, err := strconv.Atoi(lastPageStr)
	if err != nil {
		http.Error(w, "invalid page number", http.StatusBadRequest)
		return
	}

	pageIndex, err := s.getNextPageIndex(dashboard, lastPage, action)
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to get next page index: %s", err), http.StatusInternalServerError)
		return
	}

	if _, err = w.Write([]byte(fmt.Sprintf("%d", pageIndex))); err != nil {
		Error(r.Context(), w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func (s *server) getPage(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "getPage")
	dashboard := r.PathValue("dashboard")
	pageIndexStr := r.PathValue("page")
	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		http.Error(w, "invalid page number", http.StatusBadRequest)
		return
	}

	page, err := s.loadPage(dashboard, pageIndex)
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to get next page: %s", err), http.StatusInternalServerError)
		return
	}

	image, err := s.renderPage(r.Context(), *page, RenderOptions{
		BinaryPath: "wkhtmltoimage",
		AssetsPath: "./assets",
		Width:      800,
		Height:     480,
		Quality:    100,
	})
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to render page: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	if _, err = io.Copy(w, image); err != nil {
		Error(r.Context(), w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func Error(ctx context.Context, w http.ResponseWriter, error string, code int) {
	slog.ErrorContext(ctx, error)
	http.Error(w, error, code)
}
