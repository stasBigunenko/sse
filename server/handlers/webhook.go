package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"sse/models"
	"sse/service"
)

type WebhookHandler struct {
	service *service.Service

	Notifier       chan []byte                        // Events are pushed to this channel by the main events-gathering routine
	newClients     chan map[uuid.UUID]chan []byte     // New client connections are pushed to this channel
	closingClients chan map[uuid.UUID]chan []byte     // Closed client connections are pushed to this channel
	clients        map[uuid.UUID]map[chan []byte]bool // Client connections registry
	historyBuffer  map[uuid.UUID][]*models.EventMsg   // Buffer to store all sent messages
	historyMutex   sync.Mutex                         // Mutex to protect access to historyBuffer
	unsentMsg      map[uuid.UUID][]*models.EventMsg   // msg that should wait for the previous order status
}

func NewWebhookHandler(s *service.Service) *WebhookHandler {
	wh := &WebhookHandler{
		service: s,

		Notifier:       make(chan []byte),
		newClients:     make(chan map[uuid.UUID]chan []byte),
		closingClients: make(chan map[uuid.UUID]chan []byte),
		clients:        make(map[uuid.UUID]map[chan []byte]bool),
		historyBuffer:  make(map[uuid.UUID][]*models.EventMsg),
		unsentMsg:      make(map[uuid.UUID][]*models.EventMsg),
	}

	// Set it running - listening and broadcasting events
	go wh.listen()

	return wh
}

func (h *WebhookHandler) listen() {
	for {
		select {
		case client := <-h.newClients:
			for orderID, ch := range client {
				if _, exists := h.clients[orderID]; !exists {
					h.clients[orderID] = make(map[chan []byte]bool)
				}
				h.clients[orderID][ch] = true
				log.Printf("Client added for order %s. %d registered clients", orderID, len(h.clients[orderID]))
			}
		case client := <-h.closingClients:
			for orderID, ch := range client {
				delete(h.clients[orderID], ch)
				log.Printf("Removed client for order %s. %d registered clients", orderID, len(h.clients[orderID]))
			}
		case event := <-h.Notifier:
			var eventMsg models.EventBody
			err := json.Unmarshal(event, &eventMsg)
			if err != nil {
				log.Printf("Error unmarshalling event: %v", err)
				continue
			}
			orderID, err := uuid.Parse(eventMsg.OrderID)
			if err != nil {
				continue
			}
			if clients, exists := h.clients[orderID]; exists {
				for clientMessageChan := range clients {
					clientMessageChan <- event
				}
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
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Each connection registers its own message channel
	messageChan := make(chan []byte)

	// Signal that we have a new connection
	h.newClients <- map[uuid.UUID]chan []byte{orderID: messageChan}

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		h.closingClients <- map[uuid.UUID]chan []byte{orderID: messageChan}
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	historyEvents, err := h.service.GetEventHistory(r.Context(), orderID)
	if err != nil {
		SendHTTPError(w, r, err)
		return
	}

	var lastSentMessage *models.EventMsg

	lastSentMessage, err = h.sendMsg(w, flusher, historyEvents, orderID, lastSentMessage) // sent messages to the new connected client
	if err != nil {
		SendHTTPError(w, r, err)
		return
	}

	inactivityTimeout := time.NewTimer(1 * time.Minute) // start timer for close the connection
	defer inactivityTimeout.Stop()

	for {
		select {
		// Listen to connection close and un-register messageChan
		case <-r.Context().Done():
			// remove this client from the map of connected clients
			h.closingClients <- map[uuid.UUID]chan []byte{orderID: messageChan}
			return

		// Listen for incoming messages from messageChan
		case msg := <-messageChan:
			var eventMsg models.EventMsg
			if err := json.Unmarshal(msg, &eventMsg); err != nil {
				SendHTTPError(w, r, err)
				return
			}

			// Reset the timeout timer
			if !inactivityTimeout.Stop() {
				<-inactivityTimeout.C
			}
			inactivityTimeout.Reset(1 * time.Minute)

			lastSentMessage, err = h.sendMsg(w, flusher, []models.EventMsg{eventMsg}, orderID, lastSentMessage)
			if err != nil {
				SendHTTPError(w, r, err)
				return
			}

			continue

			// Timeout after 1 minute of inactivity
		case <-inactivityTimeout.C:
			h.closingClients <- map[uuid.UUID]chan []byte{orderID: messageChan}
			return
		}
	}
}

func (h *WebhookHandler) BroadcastMessage(w http.ResponseWriter, r *http.Request) {
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

	if err = h.service.AddEvent(r.Context(), event, req.OrderStatus); err != nil {
		SendHTTPError(w, r, err)
		return
	}

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

func (h *WebhookHandler) sendMsg(
	w http.ResponseWriter, flusher http.Flusher,
	events []models.EventMsg, orderID uuid.UUID,
	lastSentMessage *models.EventMsg,
) (*models.EventMsg, error) {

	for _, eventMsg := range events {
		if allowToSendMsgToStream(lastSentMessage, &eventMsg) {
			formattedMsg := fmt.Sprintf("%s\n\n", eventMsg)
			_, err := w.Write([]byte(formattedMsg))
			if err != nil {
				return nil, err
			}
			flusher.Flush()

			lastSentMessage = &eventMsg

			lastSentMessage, err = h.checkUnsentMsgToSend(lastSentMessage, orderID, w, flusher)
			if err != nil {
				return nil, err
			}
			continue
		}

		h.storeUnSentMsg(&eventMsg, orderID)
	}

	return lastSentMessage, nil
}

func (h *WebhookHandler) storeUnSentMsg(msg *models.EventMsg, orderID uuid.UUID) {
	h.historyMutex.Lock()
	defer h.historyMutex.Unlock()

	if _, exists := h.unsentMsg[orderID]; !exists {
		h.unsentMsg[orderID] = []*models.EventMsg{}
	}

	h.unsentMsg[orderID] = append(h.unsentMsg[orderID], msg)
	sort.Slice(h.unsentMsg[orderID], func(i, j int) bool {
		return h.unsentMsg[orderID][i].UpdatedAt.After(h.unsentMsg[orderID][j].UpdatedAt)
	})
}

func (h *WebhookHandler) checkUnsentMsgToSend(lastSentMessage *models.EventMsg, orderID uuid.UUID, w http.ResponseWriter, flusher http.Flusher) (*models.EventMsg, error) {
	l := len(h.unsentMsg[orderID])
	for ; l > 0; l-- {
		if allowToSendMsgToStream(lastSentMessage, h.unsentMsg[orderID][l-1]) {
			msg, err := json.Marshal(h.unsentMsg[orderID][l-1])
			if err != nil {
				return nil, err
			}

			formattedMsg := fmt.Sprintf("%s\n\n", msg)
			_, err = w.Write([]byte(formattedMsg))
			if err != nil {
				return nil, err
			}
			flusher.Flush()

			lastSentMessage = h.unsentMsg[orderID][l-1]

			h.unsentMsg[orderID] = h.unsentMsg[orderID][:l-1]
		} else {
			break
		}
	}

	return lastSentMessage, nil
}

func allowToSendMsgToStream(lastSentMsg, eventMsg *models.EventMsg) bool {
	if eventMsg.OrderStatus == models.CoolOrderCreated && lastSentMsg == nil {
		return true
	}

	if lastSentMsg != nil {
		if (eventMsg.OrderStatus == models.SBUVarificationPending && lastSentMsg.OrderStatus == models.CoolOrderCreated) ||
			canceledOrder(eventMsg.OrderStatus) {
			return true
		}

		if (eventMsg.OrderStatus == models.ConfirmedByMayor && lastSentMsg.OrderStatus == models.SBUVarificationPending) ||
			canceledOrder(eventMsg.OrderStatus) {
			return true
		}

		if lastSentMsg.OrderStatus == models.ConfirmedByMayor && (eventMsg.OrderStatus == models.Chinazes ||
			canceledOrder(eventMsg.OrderStatus)) {
			return true
		}

		if lastSentMsg.OrderStatus == models.Chinazes && eventMsg.OrderStatus == models.GiveMyMoneyBack {
			return true
		}
	}

	return false
}

func canceledOrder(orderStatus string) bool {
	return orderStatus == models.Failed || orderStatus == models.ChangedMyMind
}
