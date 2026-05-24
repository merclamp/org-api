package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/merclamp/org-api/internal/middleware"
)

func NewRouter(
	deptHandler *DepartmentHandler,
	empHandler *EmployeeHandler,
	log *slog.Logger,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{Status: "ok"})
	})

	mux.HandleFunc("/departments/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/departments/" && r.Method == http.MethodPost {
			deptHandler.Create(w, r)
			return
		}

		if strings.HasSuffix(path, "/employees/") || strings.HasSuffix(path, "/employees") {
			if r.Method == http.MethodPost {
				empHandler.Create(w, r)
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		switch r.Method {
		case http.MethodGet:
			deptHandler.Get(w, r)
		case http.MethodPatch:
			deptHandler.Update(w, r)
		case http.MethodDelete:
			deptHandler.Delete(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	return middleware.Logger(log)(mux)
}