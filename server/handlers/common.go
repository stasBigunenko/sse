package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sse/models"
)

func sendEmptyResponse(w http.ResponseWriter, r *http.Request, statusCode int) {
	log.Println(
		"resp %s: %s -%d",
		r.Method,
		r.RequestURI,
		statusCode,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
}

func sendResponse(w http.ResponseWriter, r *http.Request, statusCode int, resp interface{}) {
	log.Println(
		"resp %s: %s - %d - %v",
		r.Method,
		r.RequestURI,
		statusCode,
		resp,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	respBody, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if resp != nil {
		if _, err = w.Write(respBody); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

func SendOK(w http.ResponseWriter, r *http.Request) {
	sendEmptyResponse(w, r, http.StatusOK)
}

func SendBadRequest(w http.ResponseWriter, r *http.Request, err error) {
	sendResponse(w, r, http.StatusBadRequest, map[string]string{"message": err.Error()})
}

func SendGone(w http.ResponseWriter, r *http.Request) {
	sendEmptyResponse(w, r, http.StatusGone)
}

func SendConflict(w http.ResponseWriter, r *http.Request) {
	sendEmptyResponse(w, r, http.StatusConflict)
}

func SendInternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	sendResponse(w, r, http.StatusInternalServerError, map[string]string{"message": err.Error()})
}

func SendHTTPError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, models.ErrAlreadyExistsFinalStatus):
		SendGone(w, r)
	case errors.Is(err, models.ErrAlreadyProcessed):
		SendConflict(w, r)
	default:
		SendInternalServerError(w, r, err)
	}
}
