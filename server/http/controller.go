package http

import (
	"github.com/gorilla/mux"
	"net/http"

	"sse/server/handlers"
)

type Controller struct {
	router *mux.Router

	wh *handlers.WebhookHandler
	o  *handlers.OrdersHandler
}

func NewController(wh *handlers.WebhookHandler, o *handlers.OrdersHandler) *Controller {
	r := &Controller{
		router: mux.NewRouter(),

		wh: wh,
		o:  o,
	}

	r.initRoutes()

	return r
}

func (c *Controller) initRoutes() {
	c.router.Use(mux.CORSMethodMiddleware(c.router))

	c.router.HandleFunc("/webhooks/payments/orders", c.wh.BroadcastMessage).Methods(http.MethodPost)
	c.router.HandleFunc("/orders/{order_id}/events", c.wh.Stream).Methods(http.MethodGet)

	c.router.HandleFunc("/orders", c.o.GetOrdersByFilter).Methods(http.MethodGet)
}
