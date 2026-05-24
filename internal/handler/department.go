package handler

import (
	"encoding/json"
	"net/http"

	"github.com/merclamp/org-api/internal/domain"
	"github.com/merclamp/org-api/internal/service"
)

type DepartmentService interface {
	Create(input service.CreateDepartmentInput) (*domain.Department, error)
	Get(id int, input service.GetDepartmentInput) (*domain.Department, error)
	Update(id int, input service.UpdateDepartmentInput) (*domain.Department, error)
	Delete(id int, input service.DeleteDepartmentInput) error
}

type DepartmentHandler struct {
	svc DepartmentService
}

func NewDepartmentHandler(svc DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{svc: svc}
}

type createDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *int   `json:"parent_id"`
}

func (h *DepartmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	dept, err := h.svc.Create(service.CreateDepartmentInput{
		Name:     req.Name,
		ParentID: req.ParentID,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, dept)
}

func (h *DepartmentHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "/departments/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	depth := queryInt(r, "depth", service.DefaultDepth)
	includeEmployees := queryBool(r, "include_employees", true)

	dept, err := h.svc.Get(id, service.GetDepartmentInput{
		Depth:            depth,
		IncludeEmployees: includeEmployees,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dept)
}

type updateDepartmentRequest struct {
	Name     *string          `json:"name"`
	ParentID json.RawMessage  `json:"parent_id"`
}

func (h *DepartmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "/departments/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req updateDepartmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	input := service.UpdateDepartmentInput{
		Name: req.Name,
	}

	if len(req.ParentID) > 0 {
		if string(req.ParentID) == "null" {
			input.ClearParent = true
		} else {
			var parentID int
			if err := json.Unmarshal(req.ParentID, &parentID); err != nil {
				writeError(w, http.StatusBadRequest, "parent_id must be an integer or null")
				return
			}
			input.ParentID = &parentID
		}
	}

	dept, err := h.svc.Update(id, input)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dept)
}

func (h *DepartmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "/departments/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	mode := queryString(r, "mode")
	if mode == "" {
		writeError(w, http.StatusBadRequest, "mode is required: cascade or reassign")
		return
	}

	reassignTo, err := queryIntPtr(r, "reassign_to_department_id")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.Delete(id, service.DeleteDepartmentInput{
		Mode:                   mode,
		ReassignToDepartmentID: reassignTo,
	}); err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}