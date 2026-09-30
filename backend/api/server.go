package api

import (
	"context"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: handler,
			// ReadTimeout ATAYLAB yo'q: u BUTUN tanani o'qishga qo'llanadi va sekin
			// internetdagi 21 MB chat upload'ini 15s da o'ldirardi. Slowloris'dan
			// himoyani ReadHeaderTimeout beradi; tana hajmi MaxBytesReader bilan cheklangan.
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
