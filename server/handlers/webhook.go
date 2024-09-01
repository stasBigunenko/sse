package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"sse/models"
	"sse/service"
)

type WebhookHandler struct {
	service *service.Service

	// Events are pushed to this channel by the main events-gathering routine
	Notifier chan []byte
	// New client connections are pushed to this channel
	newClients chan chan []byte
	// Closed client connections are pushed to this channel
	closingClients chan chan []byte
	// Client connections registry
	clients map[chan []byte]bool
}

func NewWebhookHandler(s *service.Service) *WebhookHandler {
	wh := &WebhookHandler{
		service: s,

		Notifier:       make(chan []byte),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	// Set it running - listening and broadcasting events
	go wh.listen()

	return wh
}

func (h *WebhookHandler) listen() {
	for {
		select {
		case client := <-h.newClients:
			// A new client has connected.
			// Register their message channel
			h.clients[client] = true
			log.Printf("Client added. %d registered clients", len(h.clients))
		case client := <-h.closingClients:
			// A client has dettached and we want to
			// stop sending them messages.
			delete(h.clients, client)
			log.Printf("Removed client. %d registered clients", len(h.clients))
		case event := <-h.Notifier:
			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan := range h.clients {
				clientMessageChan <- event
			}
		}
	}
}

func (h *WebhookHandler) Stream(w http.ResponseWriter, r *http.Request) {
	orderID, err := uuid.Parse(mux.Vars(r)["order_id"])
	if err != nil {
		SendHTTPError(w, r, err)
		return
	}
	// Check if the ResponseWriter supports flushing.
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection
	h.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		h.closingClients <- messageChan
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// If the stream is finished, send the history to the new client
	completed, err := h.service.CheckCompletedOrderStatusByID(r.Context(), orderID)
	if err != nil {
		SendHTTPError(w, r, err)
		return
	}

	if completed {
		h.sendEventHistory(w, flusher, r, orderID)
		return
	}

	// Timer to close the connection after 1 minute of inactivity
	inactivityTimeout := time.NewTimer(10 * time.Minute)
	defer inactivityTimeout.Stop()

	var completedTimer *time.Timer

	for {
		select {
		// Listen to connection close and un-register messageChan
		case <-r.Context().Done():
			// remove this client from the map of connected clients
			h.closingClients <- messageChan
			return

		// Listen for incoming messages from messageChan
		case msg := <-messageChan:
			if err = h.handleMessage(r.Context(), msg, &completedTimer); err != nil {
				SendHTTPError(w, r, err)
				return
			}

			// Reset the timeout timer
			if !inactivityTimeout.Stop() {
				<-inactivityTimeout.C
			}
			inactivityTimeout.Reset(10 * time.Minute)

			formattedMsg := fmt.Sprintf("data: %s\n\n", msg)
			_, err := w.Write([]byte(formattedMsg))
			if err != nil {
				SendHTTPError(w, r, err)
				return
			}
			flusher.Flush()
			flusher.Flush()

			// Timeout after 1 minute of inactivity
		case <-inactivityTimeout.C:
			h.closingClients <- messageChan
			if err = h.finishStream(r.Context(), orderID); err != nil {
				SendHTTPError(w, r, err)
				return
			}
			return

		case <-h.getCompletedTimerChan(completedTimer):
			if err = h.finishStream(r.Context(), orderID); err != nil {
				SendHTTPError(w, r, err)
				return
			}
			return
		}
	}
}

func (h *WebhookHandler) BroadcastMessage(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req models.EventBody
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	event, err := h.validateEventReq(req)
	if err != nil {
		SendInternalServerError(w, r, err)
		return
	}

	completed, err := h.service.CheckCompletedOrderStatusByID(r.Context(), event.OrderID)
	if err != nil {
		SendInternalServerError(w, r, err)
		return
	}

	if completed {
		SendGone(w, r)
		return
	}

	if err = h.service.AddEvent(r.Context(), event, req.OrderStatus); err != nil {
		SendHTTPError(w, r, err)
		return
	}

	// Send the message to the broker via Notifier channel
	j, err := json.Marshal(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Notifier <- []byte(j)

	SendOK(w, r)
}

func (h *WebhookHandler) validateEventReq(req models.EventBody) (models.Event, error) {
	eventID, err := uuid.Parse(req.EventID)
	if err != nil {
		return models.Event{}, err
	}

	orderID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return models.Event{}, err
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return models.Event{}, err
	}

	if req.OrderStatus == models.EmptyString {
		return models.Event{}, models.ErrBadRequest
	}

	createdAt, err := time.Parse(models.TimeFormat, req.CreatedAt)
	if err != nil {
		return models.Event{}, err

	}

	updatedAt, err := time.Parse(models.TimeFormat, req.UpdatedAt)
	if err != nil {
		return models.Event{}, err

	}

	return models.Event{
		EventID:       eventID,
		OrderID:       orderID,
		UserID:        userID,
		OrderStatusID: 0,
		UpdatedAt:     updatedAt,
		CreatedAt:     createdAt,
	}, nil
}

func (h *WebhookHandler) handleMessage(ctx context.Context, msg []byte, completedTimer **time.Timer) error {
	var eventReq models.EventBody
	if err := json.Unmarshal(msg, &eventReq); err != nil {
		return err
	}

	orderID, err := uuid.Parse(eventReq.OrderID)
	if err != nil {
		return err
	}

	orderStatus, err := h.service.GetOrderStatusByName(ctx, eventReq.OrderStatus)
	if err != nil {
		return err
	}

	if orderStatus.IsFinal {
		switch eventReq.OrderStatus {
		case models.Chinazes:
			if *completedTimer != nil {
				(*completedTimer).Stop()
			}
			*completedTimer = time.NewTimer(3000 * time.Second)
		case models.GiveMyMoneyBack:
			if *completedTimer != nil {
				(*completedTimer).Stop()
				*completedTimer = nil

				if err = h.finishStream(ctx, orderID); err != nil {
					return err
				}
			}
		default:
			if err = h.finishStream(ctx, orderID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *WebhookHandler) sendEventHistory(w http.ResponseWriter, flusher http.Flusher, r *http.Request, orderID uuid.UUID) {
	events, err := h.service.GetEventHistory(r.Context(), orderID)
	if err != nil {
		http.Error(w, "Failed to retrieve event history", http.StatusInternalServerError)
		return
	}

	for _, event := range events {
		eventData, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal event: %v", err)
			continue
		}

		w.Write(eventData)
		flusher.Flush()
	}
}

func (h *WebhookHandler) finishStream(ctx context.Context, orderID uuid.UUID) error {
	if err := h.service.AddCompletedOrder(ctx, orderID); err != nil {
		return err
	}

	return nil
}

func (h *WebhookHandler) getCompletedTimerChan(timer *time.Timer) <-chan time.Time {
	if timer != nil {
		return timer.C
	}
	return nil
}
