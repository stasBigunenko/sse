package http

import (
	"context"
	"log"
	"net/http"
	"time"

	"sse/config"
)

type Server struct {
	server     *http.Server
	controller *Controller
}

func NewHTPPServer(controller *Controller, cfgHTTP config.HTTPServerConfig) *Server {
	return &Server{
		controller: controller,
		server: &http.Server{
			Addr:    ":" + cfgHTTP.Port,
			Handler: controller.router,
		},
	}
}

func (s *Server) Run(ctx context.Context) {
	go func() {
		log.Printf("Server starts on port: %s\n", s.server.Addr)

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
