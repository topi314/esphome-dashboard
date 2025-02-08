package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	r := http.NewServeMux()
	s := newServer("dashboards", ":8080", r)

	r.HandleFunc("GET /dashboards/{dashboard}/control", s.getControl)
	r.HandleFunc("GET /dashboards/{dashboard}/pages/{page}", s.getPage)

	s.Start()

	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM)
	<-si
}
