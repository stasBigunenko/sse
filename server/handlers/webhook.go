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

type clientState struct {
	messageChan     chan []byte
	lastSentMessage *models.EventMsg
	unsentMsg       []*models.EventMsg
}

type WebhookHandler struct {
	service *service.Service

	Notifier       chan []byte                         // Events are pushed to this channel by the main events-gathering routine
	newClients     chan map[uuid.UUID]chan []byte      // New client connections are pushed to this channel
	closingClients chan map[uuid.UUID]chan []byte      // Closed client connections are pushed to this channel
	clients        map[uuid.UUID]map[*clientState]bool // Client connections registry
	clientsMutex   sync.Mutex                          // Mutex to protect access to clients map
}

func NewWebhookHandler(s *service.Service) *WebhookHandler {
	wh := &WebhookHandler{
		service: s,

		Notifier:       make(chan []byte),
		newClients:     make(chan map[uuid.UUID]chan []byte),
		closingClients: make(chan map[uuid.UUID]chan []byte),
		clients:        make(map[uuid.UUID]map[*clientState]bool),
	}

	go wh.listen()

	return wh
}

func (h *WebhookHandler) listen() {
	for {
		select {
		case client := <-h.newClients:
			h.clientsMutex.Lock()
			for orderID, ch := range client {
				if _, exists := h.clients[orderID]; !exists {
					h.clients[orderID] = make(map[*clientState]bool)
				}
				h.clients[orderID][&clientState{messageChan: ch}] = true
				log.Printf("Client added for order %s. %d registered clients", orderID, len(h.clients[orderID]))
			}
			h.clientsMutex.Unlock()

		case client := <-h.closingClients:
			h.clientsMutex.Lock()
			for orderID, ch := range client {
				delete(h.clients[orderID], &clientState{messageChan: ch})
				log.Printf("Removed client for order %s. %d registered clients", orderID, len(h.clients[orderID]))
			}
			h.clientsMutex.Unlock()

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

			h.clientsMutex.Lock()
			if clients, exists := h.clients[orderID]; exists {
				for client := range clients {
					select {
					case client.messageChan <- event:
					default:
						// Avoid blocking if the client is slow
					}
				}
			}
			h.clientsMutex.Unlock()
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

	client := &clientState{
		messageChan:     make(chan []byte, 5),
		lastSentMessage: nil,
	}

	h.newClients <- map[uuid.UUID]chan []byte{orderID: client.messageChan}

	defer func() {
		h.closingClients <- map[uuid.UUID]chan []byte{orderID: client.messageChan}
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

	err = h.sendMsg(w, flusher, historyEvents, client)
	if err != nil {
		SendHTTPError(w, r, err)
		return
	}

	inactivityTimeout := time.NewTimer(models.InactivityTimeout)
	defer inactivityTimeout.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.closingClients <- map[uuid.UUID]chan []byte{orderID: client.messageChan}
			return

		case msg := <-client.messageChan:
			inactivityTimeout.Reset(models.InactivityTimeout)

			var eventMsg models.EventMsg
			if err := json.Unmarshal(msg, &eventMsg); err != nil {
				SendHTTPError(w, r, err)
				return
			}

			if err = h.sendMsg(w, flusher, []models.EventMsg{eventMsg}, client); err != nil {
				SendHTTPError(w, r, err)
				return
			}

		case <-inactivityTimeout.C:
			h.closingClients <- map[uuid.UUID]chan []byte{orderID: client.messageChan}
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

	if len(req.OrderStatus) == 0 {
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
	events []models.EventMsg,
	client *clientState,
) error {

	for _, eventMsg := range events {
		if allowToSendMsgToStream(client.lastSentMessage, &eventMsg) {
			formattedMsg := fmt.Sprintf("%s\n\n", eventMsg)
			_, err := w.Write([]byte(formattedMsg))
			if err != nil {
				return err
			}
			flusher.Flush()

			client.lastSentMessage = &eventMsg

			if err = client.checkUnsentMsgToSend(w, flusher); err != nil {
				return err
			}
			continue
		}

		client.storeUnSentMsg(&eventMsg)
	}

	return nil
}

func (c *clientState) storeUnSentMsg(msg *models.EventMsg) {
	c.unsentMsg = append(c.unsentMsg, msg)
	sort.Slice(c.unsentMsg, func(i, j int) bool {
		return c.unsentMsg[i].UpdatedAt.After(c.unsentMsg[j].UpdatedAt)
	})
}

func (c *clientState) checkUnsentMsgToSend(w http.ResponseWriter, flusher http.Flusher) error {
	l := len(c.unsentMsg)
	for ; l > 0; l-- {
		if allowToSendMsgToStream(c.lastSentMessage, c.unsentMsg[l-1]) {
			msg, err := json.Marshal(c.unsentMsg[l-1])
			if err != nil {
				return err
			}

			formattedMsg := fmt.Sprintf("%s\n\n", msg)
			_, err = w.Write([]byte(formattedMsg))
			if err != nil {
				return err
			}
			flusher.Flush()

			c.lastSentMessage = c.unsentMsg[l-1]

			c.unsentMsg = c.unsentMsg[:l-1]
		} else {
			break
		}
	}

	return nil
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
