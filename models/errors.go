package models

import "errors"

const EmptyString = ""

var (
	ErrBadRequest               = errors.New("bad request")
	ErrInternalServer           = errors.New("internal server")
	ErrAlreadyExistsFinalStatus = errors.New("already exists final status of the order")
	ErrAlreadyProcessed         = errors.New("event already processed")
)
