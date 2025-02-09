package dashboard

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

func (s *Server) getControl(w http.ResponseWriter, r *http.Request) {
	dashboard := r.PathValue("dashboard")

	query := r.URL.Query()
	action := Action(query.Get("action"))
	lastPageStr := query.Get("page")

	slog.InfoContext(r.Context(), "getControl", slog.String("dashboard", dashboard), slog.String("action", string(action)), slog.String("last_page", lastPageStr))

	lastPage, err := strconv.Atoi(lastPageStr)
	if err != nil {
		Error(r.Context(), w, "invalid page number", http.StatusBadRequest)
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

func (s *Server) getPage(w http.ResponseWriter, r *http.Request) {
	dashboard := r.PathValue("dashboard")
	pageIndexStr := r.PathValue("page")

	query := r.URL.Query()
	htmlStr := query.Get("html")

	slog.InfoContext(r.Context(), "getPage", slog.String("dashboard", dashboard), slog.String("page", pageIndexStr), slog.String("html", htmlStr))

	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		Error(r.Context(), w, "invalid page number", http.StatusBadRequest)
		return
	}

	var html bool
	if htmlStr != "" {
		html, err = strconv.ParseBool(htmlStr)
		if err != nil {
			Error(r.Context(), w, "invalid html flag", http.StatusBadRequest)
			return
		}
	}

	base, err := s.loadDashboard(dashboard, pageIndex)
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to get next page: %s", err), http.StatusInternalServerError)
		return
	}

	content, contentLength, err := s.renderDashboard(r.Context(), *base, RenderOptions{
		Width:     base.Config.Width,
		Height:    base.Config.Height,
		PrintHTML: html,
	})
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to render page: %s", err), http.StatusInternalServerError)
		return
	}

	if html {
		w.Header().Set("Content-Type", "text/html")
	} else {
		w.Header().Set("Content-Type", "image/png")
	}
	w.Header().Set("Content-Length", strconv.Itoa(contentLength))
	if _, err = io.Copy(w, content); err != nil {
		Error(r.Context(), w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func Error(ctx context.Context, w http.ResponseWriter, error string, code int) {
	slog.ErrorContext(ctx, error)
	http.Error(w, error, code)
}
