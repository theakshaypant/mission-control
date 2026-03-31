// Package api provides the HTTP API server for mission-control.
// It exposes item queries and state mutations over a minimal REST interface,
// using only the stdlib net/http package (Go 1.22+ method+path mux patterns).
package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/theakshaypant/mission-control/internal/actions"
)

// Server is the mission-control HTTP API server.
type Server struct {
	srv *http.Server
}

// New constructs a Server that listens on addr and routes all requests
// through the actions layer.
func New(addr string, a *actions.Actions) *Server {
	items := newItemsHandler(a)
	sync := newSyncHandler(a)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /items", items.list)
	mux.HandleFunc("GET /summary", items.summary)
	mux.HandleFunc("POST /items/{id}/dismiss", items.dismiss)
	mux.HandleFunc("POST /items/{id}/snooze", items.snooze)
	mux.HandleFunc("GET /sync/status", sync.status)
	mux.HandleFunc("POST /sync", sync.syncAll)
	mux.HandleFunc("POST /sync/{source}", sync.syncSource)

	return &Server{
		srv: &http.Server{
			Addr:    addr,
			Handler: chain(mux, withCORS, withRecovery),
		},
	}
}

// ServeHTTP implements http.Handler, enabling use with httptest.NewServer and
// direct handler testing without starting a real listener.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.srv.Handler.ServeHTTP(w, r)
}

// ListenAndServe starts the server and blocks until ctx is cancelled or the
// server encounters a fatal error. On context cancellation it performs a
// graceful shutdown with a 10-second deadline.
func (s *Server) ListenAndServe(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- s.srv.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
