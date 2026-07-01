package httpserver

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"time"
)

func StartHTTPServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Print("Stopping HTTP server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Failed to shutdown HTTP server gracefully: %v", err)
		}
	}()

	log.Print("Listening for HTTP traffic at 0.0.0.0:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Failed to start HTTP server: %v", err)
	}
	log.Print("Stopped HTTP server")
}
