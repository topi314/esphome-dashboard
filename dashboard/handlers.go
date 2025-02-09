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

	slog.InfoContext(r.Context(), "getPage", slog.String("dashboard", dashboard), slog.String("page", pageIndexStr))

	pageIndex, err := strconv.Atoi(pageIndexStr)
	if err != nil {
		Error(r.Context(), w, "invalid page number", http.StatusBadRequest)
		return
	}

	page, err := s.loadPage(dashboard, pageIndex)
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to get next page: %s", err), http.StatusInternalServerError)
		return
	}

	pagePNG, pagePNGLen, err := s.renderPage(r.Context(), *page, RenderOptions{
		BinaryPath: s.cfg.WKHTMLToImagePath,
		AssetsDir:  page.DashboardConfig.AssetsDir,
		Width:      page.DashboardConfig.Width,
		Height:     page.DashboardConfig.Height,
		Quality:    page.DashboardConfig.Quality,
	})
	if err != nil {
		Error(r.Context(), w, fmt.Sprintf("failed to render page: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", pagePNGLen))
	if _, err = io.Copy(w, pagePNG); err != nil {
		Error(r.Context(), w, "failed to write response", http.StatusInternalServerError)
		return
	}
}

func Error(ctx context.Context, w http.ResponseWriter, error string, code int) {
	slog.ErrorContext(ctx, error)
	http.Error(w, error, code)
}
