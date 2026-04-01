// Package api provides the HTTP API server for mission-control.
// It exposes item queries and state mutations over a minimal REST interface,
// using only the stdlib net/http package (Go 1.22+ method+path mux patterns).
package api

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"time"

	"github.com/theakshaypant/mission-control/internal/actions"
)

// Server is the mission-control HTTP API server.
type Server struct {
	srv *http.Server
}

// New constructs a Server that listens on addr and routes all requests
// through the actions layer. If static is non-nil it is served at / as a
// single-page application (unknown paths fall back to index.html).
func New(addr string, a *actions.Actions, static fs.FS) *Server {
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

	if static != nil {
		mux.Handle("/", spaHandler(static))
	}

	return &Server{
		srv: &http.Server{
			Addr:    addr,
			Handler: chain(mux, withCORS, withRecovery),
		},
	}
}

// spaHandler serves files from the given FS. If the requested path does not
// exist in the FS it falls back to index.html, supporting client-side routing.
func spaHandler(files fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check whether the file exists in the embedded FS.
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		}
		// Strip leading slash for fs.Stat.
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if _, err := fs.Stat(files, path); err != nil {
			// File not found — serve index.html for SPA routing.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
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
