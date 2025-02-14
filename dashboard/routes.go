package dashboard

import (
	"net/http"
)

func (s *Server) Routes() http.Handler {
	r := http.NewServeMux()

	r.HandleFunc("GET /version", s.getVersion)
	r.HandleFunc("GET /dashboards/{dashboard}/control", s.getControl)
	r.HandleFunc("GET /dashboards/{dashboard}/pages/{page}", s.getPage)
	r.HandleFunc("GET /dashboards/{dashboard}/assets/", s.getAsset)

	return r
}
