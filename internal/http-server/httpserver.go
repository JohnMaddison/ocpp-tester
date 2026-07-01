package httpserver

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/controller"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

func StartHTTPServer(ctx context.Context, sessionsService *service.SessionsService, ocpp16Service *service.OCPP16Service, ocppLogService *service.OCPPLogService) {
	mux := http.NewServeMux()
	controller.NewSessionsController(sessionsService).RegisterRoutes(mux)
	controller.NewOCPP16Controller(ocpp16Service).RegisterRoutes(mux)
	controller.NewOCPPLogController(ocppLogService).RegisterRoutes(mux)

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

	log.Print("Listening for HTTP traffic at http://0.0.0.0:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Failed to start HTTP server: %v", err)
	}
	log.Print("Stopped HTTP server")
}
