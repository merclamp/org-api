package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/merclamp/org-api/internal/domain"
)

type errorResponse struct {
	Error string `json:"error"`
}

type statusResponse struct {
	Status string `json:"status"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrCyclicReference):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		slog.Error("internal error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}