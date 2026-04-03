package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	readTimeout     = 10 * time.Second
	writeTimeout    = 30 * time.Second
	idleTimeout     = 120 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Server wraps an http.Server with graceful shutdown support.
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new Server bound to the given port with the provided handler.
func NewServer(port string, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         net.JoinHostPort("", port),
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}
}

// Run starts the HTTP server and blocks until a shutdown signal is received.
// It performs a graceful shutdown, allowing in-flight requests to complete.
func (s *Server) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)

	go func() {
		log.Printf("server listening on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Println("shutdown signal received, draining connections...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("server stopped gracefully")
	return nil
}
