package handlers

import (
	"net/http"
	"sse/models"
	"sse/service"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type OrdersHandler struct {
	service *service.Service
}

func NewOrdersHandler(s *service.Service) *OrdersHandler {
	return &OrdersHandler{service: s}
}

func (h *OrdersHandler) GetOrdersByFilter(w http.ResponseWriter, r *http.Request) {
	filters, err := parseOrdersFilters(r)
	if err != nil {
		SendBadRequest(w, r, err)
		return
	}

	res, err := h.service.GetOrders(r.Context(), filters)
	if err != nil {
		SendHTTPError(w, r, err)
		return
	}

	sendResponse(w, r, http.StatusOK, res)
}

func parseOrdersFilters(r *http.Request) (*models.OrderFilter, error) {
	var (
		statuses   []string
		isFinalPtr *bool
		userID     uuid.UUID
		limit      int
		offset     int
		sortBy     string
		sortOrder  string

		err error
	)
	statusesStr := r.URL.Query().Get("status")
	isFinalStr := r.URL.Query().Get("is_final")
	userIDStr := r.URL.Query().Get("user_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	sortByStr := r.URL.Query().Get("sort_by")
	sortOrderStr := r.URL.Query().Get("sort_order")

	if (len(statusesStr) == 0 && len(isFinalStr) == 0) ||
		(len(statusesStr) != 0 && len(isFinalStr) != 0) {
		return nil, models.ErrBadRequest
	}

	statuses, err = makeStringSlice(statusesStr)
	if err != nil {
		return nil, err
	}

	if len(isFinalStr) != 0 {
		isFinal, err := strconv.ParseBool(isFinalStr)
		if err != nil {
			return nil, err
		}
		isFinalPtr = &isFinal
	} else {
		isFinalPtr = nil
	}

	if len(userIDStr) != 0 {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, err
		}
	}

	if len(limitStr) != 0 {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}
	} else {
		limit = 10
	}

	if len(offsetStr) != 0 {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return nil, err
		}
	} else {
		offset = 0
	}

	if len(sortByStr) != 0 {
		if sortByStr != models.SortByCreatedAt && sortByStr != models.SortByUpdatedAt {
			return nil, models.ErrBadRequest
		}
		sortBy = sortByStr
	} else {
		sortBy = models.SortByCreatedAt
	}

	if len(sortOrderStr) != 0 {
		if sortOrderStr != models.OrderASC && sortOrderStr != models.OrderDESC {
			return nil, models.ErrBadRequest
		}
		sortOrder = sortOrderStr
	} else {
		sortOrder = models.OrderDESC
	}

	return &models.OrderFilter{
		Status:    statuses,
		UserID:    userID,
		Limit:     limit,
		Offset:    offset,
		IsFinal:   isFinalPtr,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}, nil
}

func makeStringSlice(input string) ([]string, error) {
	if len(input) == 0 {
		return nil, nil
	}

	statuses := strings.Split(input, ",")

	return statuses, nil
}
