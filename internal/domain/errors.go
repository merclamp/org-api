package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrValidation      = errors.New("validation error")
	ErrCyclicReference = errors.New("cyclic reference detected")
)