package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/merclamp/org-api/internal/domain"
)

type CreateEmployeeInput struct {
	DepartmentID int
	FullName     string
	Position     string
	HiredAt      *string // "2024-01-15" | nil
}

type EmployeeService struct {
	empRepo  EmployeeRepo
	deptRepo DepartmentRepo
}

func NewEmployeeService(empRepo EmployeeRepo, deptRepo DepartmentRepo) *EmployeeService {
	return &EmployeeService{
		empRepo:  empRepo,
		deptRepo: deptRepo,
	}
}

func (s *EmployeeService) Create(input CreateEmployeeInput) (*domain.Employee, error) {
	if _, err := s.deptRepo.FindByID(input.DepartmentID); err != nil {
		if err == domain.ErrNotFound {
			return nil, fmt.Errorf("%w: department %d not found", domain.ErrNotFound, input.DepartmentID)
		}
		return nil, err
	}

	fullName, err := validateField(input.FullName, "full_name")
	if err != nil {
		return nil, err
	}

	position, err := validateField(input.Position, "position")
	if err != nil {
		return nil, err
	}

	hiredAt, err := parseDate(input.HiredAt)
	if err != nil {
		return nil, err
	}

	emp := &domain.Employee{
		DepartmentID: input.DepartmentID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      hiredAt,
	}

	if err := s.empRepo.Create(emp); err != nil {
		return nil, err
	}

	return emp, nil
}

func validateField(value, fieldName string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: %s cannot be empty", domain.ErrValidation, fieldName)
	}
	if len(value) > MaxNameLength {
		return "", fmt.Errorf("%w: %s cannot exceed %d characters", domain.ErrValidation, fieldName, MaxNameLength)
	}
	return value, nil
}

func parseDate(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return nil, fmt.Errorf("%w: hired_at must be in format YYYY-MM-DD", domain.ErrValidation)
	}
	return &t, nil
}