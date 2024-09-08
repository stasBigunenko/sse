package models

import "errors"

var (
	ErrBadRequest               = errors.New("bad request")
	ErrAlreadyExistsFinalStatus = errors.New("already exists final status of the order")
	ErrAlreadyProcessed         = errors.New("event already processed")
)
