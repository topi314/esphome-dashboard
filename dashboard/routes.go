package dashboard

import (
	"net/http"
)

func (s *Server) Routes() http.Handler {
	r := http.NewServeMux()

	r.HandleFunc("GET /dashboards/{dashboard}/control", s.getControl)
	r.HandleFunc("GET /dashboards/{dashboard}/pages/{page}", s.getPage)

	return r
}
