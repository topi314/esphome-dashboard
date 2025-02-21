package dashboard

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

type Action string

const (
	ActionRefresh   Action = "refresh"
	ActionNextPage  Action = "next_page"
	ActionLastPage  Action = "last_page"
	ActionPrevPage  Action = "prev_page"
	ActionFirstPage Action = "first_page"
)

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	slog.InfoContext(r.Context(), "getVersion")

	if _, err := fmt.Fprintf(w, "Dashboard %s (Go %s)", s.version, s.goVersion); err != nil {
		Error(r.Context(), w, "failed to write response", http.StatusInternalServerError)
	}
}

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
	}
}

func (s *Server) getPage(w http.ResponseWriter, r *http.Request) {
	dashboard := r.PathValue("dashboard")
	pageIndexStr := r.PathValue("page")

	query := r.URL.Query()
	format := query.Get("format")

	slog.InfoContext(r.Context(), "getPage", slog.String("dashboard", dashboard), slog.String("page", pageIndexStr), slog.String("format", format))

	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		Error(r.Context(), w, "invalid page number", http.StatusBadRequest)
		return
	}

	if format == "" {
		format = "html"
	}

	if format == "html" {
		base, err := s.loadDashboard(dashboard, pageIndex)
		if err != nil {
			Error(r.Context(), w, fmt.Sprintf("failed to get next page: %s", err), http.StatusInternalServerError)
			return
		}

		content, contentLength, err := s.executeDashboard(r.Context(), *base)
		if err != nil {
			Error(r.Context(), w, fmt.Sprintf("failed to render page: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(contentLength))
		if _, err = io.Copy(w, content); err != nil {
			slog.ErrorContext(r.Context(), "failed to write response", slog.Any("err", err))
		}
		return
	}
	config, err := s.getDashboardConfig(dashboard)
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to get dashboard config: %s", err), http.StatusInternalServerError)
		return
	}

	content, contentLength, contentType, err := s.renderDashboard(r.Context(), dashboard, pageIndex, config.Width, config.Height, format)
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to render page: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(contentLength))
	if _, err = io.Copy(w, content); err != nil {
		slog.ErrorContext(r.Context(), "failed to write response", slog.Any("err", err))
	}
}

func (s *Server) getAsset(w http.ResponseWriter, r *http.Request) {
	dashboard := r.PathValue("dashboard")
	path := strings.TrimPrefix(r.URL.Path, "/dashboards/"+dashboard+"/assets")

	slog.InfoContext(r.Context(), "getAssets", slog.String("dashboard", dashboard), slog.String("path", path))

	http.ServeFile(w, r, filepath.Join(s.cfg.DashboardDir, dashboard, "assets", path))
}

func Error(ctx context.Context, w http.ResponseWriter, error string, code int) {
	slog.ErrorContext(ctx, error)
	http.Error(w, error, code)
}
