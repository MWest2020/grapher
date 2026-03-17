package serve

import (
	"context"
	"fmt"
	"net/http"
)

// Run starts the HTTP server on the given address and blocks until ctx is done.
func Run(ctx context.Context, addr string, store *JobStore, pipeline Pipeline) error {
	mux := http.NewServeMux()
	srv := NewServer(store, pipeline)
	srv.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("grapher serve listening on %s\n", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return httpServer.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}
