package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"syscall"

	"sse/config"
	"sse/server/handlers"
	"sse/server/http"
	"sse/service"
	"sse/storage/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := config.Load()
	if err != nil {
		log.Fatal(err)
		return
	}

	dbConn, err := postgres.NewPG(ctx, config.Appconfig.Postgres)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer dbConn.Close()

	if err = dbConn.Ping(ctx); err != nil {
		log.Fatal(err)
		return
	}

	services := service.New(dbConn.NewWebhookRepo(), dbConn.NewOrdersRepo())

	wh := handlers.NewWebhookHandler(services)

	router := http.NewController(wh, handlers.NewOrdersHandler(services))
	httpSrv := http.NewHTPPServer(router, config.Appconfig.HTTPServer)

	httpSrv.Run(ctx)
}
