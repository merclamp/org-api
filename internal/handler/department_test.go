package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merclamp/org-api/internal/domain"
	"github.com/merclamp/org-api/internal/service"
)


type mockDeptSvc struct {
	createFn func(input service.CreateDepartmentInput) (*domain.Department, error)
	getFn    func(id int, input service.GetDepartmentInput) (*domain.Department, error)
	updateFn func(id int, input service.UpdateDepartmentInput) (*domain.Department, error)
	deleteFn func(id int, input service.DeleteDepartmentInput) error
}

func (m *mockDeptSvc) Create(input service.CreateDepartmentInput) (*domain.Department, error) {
	if m.createFn != nil {
		return m.createFn(input)
	}
	return &domain.Department{ID: 1, Name: input.Name, ParentID: input.ParentID}, nil
}

func (m *mockDeptSvc) Get(id int, input service.GetDepartmentInput) (*domain.Department, error) {
	if m.getFn != nil {
		return m.getFn(id, input)
	}
	return &domain.Department{ID: id, Name: "Test"}, nil
}

func (m *mockDeptSvc) Update(id int, input service.UpdateDepartmentInput) (*domain.Department, error) {
	if m.updateFn != nil {
		return m.updateFn(id, input)
	}
	return &domain.Department{ID: id, Name: "Updated"}, nil
}

func (m *mockDeptSvc) Delete(id int, input service.DeleteDepartmentInput) error {
	if m.deleteFn != nil {
		return m.deleteFn(id, input)
	}
	return nil
}

type mockEmpSvc struct{}

func (m *mockEmpSvc) Create(input service.CreateEmployeeInput) (*domain.Employee, error) {
	return &domain.Employee{ID: 1, FullName: input.FullName}, nil
}

func setupRouter(deptSvc DepartmentService, empSvc EmployeeService) http.Handler {
	deptHandler := NewDepartmentHandler(deptSvc)
	empHandler := NewEmployeeHandler(empSvc)
	return NewRouter(deptHandler, empHandler, nil)
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		err := json.NewEncoder(&buf).Encode(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	return rec
}

func TestCreateDepartmentHandler_Success(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodPost, "/departments/", map[string]any{
		"name": "Engineering",
	})

	assert.Equal(t, http.StatusCreated, rec.Code)

	var dept domain.Department
	err := json.NewDecoder(rec.Body).Decode(&dept)
	require.NoError(t, err)
	assert.Equal(t, "Engineering", dept.Name)
}

func TestCreateDepartmentHandler_EmptyBody(t *testing.T) {
	svc := &mockDeptSvc{
		createFn: func(input service.CreateDepartmentInput) (*domain.Department, error) {
			return nil, fmt.Errorf("%w: name cannot be empty", domain.ErrValidation)
		},
	}
	router := setupRouter(svc, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodPost, "/departments/", map[string]any{
		"name": "",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateDepartmentHandler_InvalidJSON(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	req := httptest.NewRequest(http.MethodPost, "/departments/", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetDepartmentHandler_Success(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodGet, "/departments/1", nil)

	assert.Equal(t, http.StatusOK, rec.Code)

	var dept domain.Department
	err := json.NewDecoder(rec.Body).Decode(&dept)
	require.NoError(t, err)
	assert.Equal(t, 1, dept.ID)
}

func TestGetDepartmentHandler_NotFound(t *testing.T) {
	svc := &mockDeptSvc{
		getFn: func(id int, input service.GetDepartmentInput) (*domain.Department, error) {
			return nil, domain.ErrNotFound
		},
	}
	router := setupRouter(svc, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodGet, "/departments/999", nil)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetDepartmentHandler_InvalidID(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodGet, "/departments/abc", nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateDepartmentHandler_Success(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodPatch, "/departments/1", map[string]any{
		"name": "New Name",
	})

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUpdateDepartmentHandler_CyclicRef(t *testing.T) {
	svc := &mockDeptSvc{
		updateFn: func(id int, input service.UpdateDepartmentInput) (*domain.Department, error) {
			return nil, domain.ErrCyclicReference
		},
	}
	router := setupRouter(svc, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodPatch, "/departments/1", map[string]any{
		"parent_id": 1,
	})

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestDeleteDepartmentHandler_Cascade(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodDelete, "/departments/1?mode=cascade", nil)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteDepartmentHandler_MissingMode(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodDelete, "/departments/1", nil)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHealthHandler(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodGet, "/health", nil)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp statusResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

func TestMethodNotAllowed(t *testing.T) {
	router := setupRouter(&mockDeptSvc{}, &mockEmpSvc{})

	rec := doRequest(t, router, http.MethodPut, "/departments/1", nil)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}