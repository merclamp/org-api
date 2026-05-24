package handler

import (
	"net/http"

	"github.com/merclamp/org-api/internal/domain"
	"github.com/merclamp/org-api/internal/service"
)

type EmployeeService interface {
	Create(input service.CreateEmployeeInput) (*domain.Employee, error)
}

type EmployeeHandler struct {
	svc EmployeeService
}

func NewEmployeeHandler(svc EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

type createEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	deptID, err := pathID(r, "/departments/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req createEmployeeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	emp, err := h.svc.Create(service.CreateEmployeeInput{
		DepartmentID: deptID,
		FullName:     req.FullName,
		Position:     req.Position,
		HiredAt:      req.HiredAt,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, emp)
}