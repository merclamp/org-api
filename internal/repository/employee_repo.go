package repository

import (
	"errors"

	"gorm.io/gorm"

	"github.com/merclamp/org-api/internal/domain"
)

type EmployeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) Create(emp *domain.Employee) error {
	result := r.db.Create(emp)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *EmployeeRepository) FindByDepartmentID(departmentID int) ([]domain.Employee, error) {
	var employees []domain.Employee
	result := r.db.Where("department_id = ?", departmentID).
		Order("full_name ASC").
		Find(&employees)
	if result.Error != nil {
		return nil, result.Error
	}
	return employees, nil
}

func (r *EmployeeRepository) FindByID(id int) (*domain.Employee, error) {
	var emp domain.Employee
	result := r.db.First(&emp, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, result.Error
	}
	return &emp, nil
}

func (r *EmployeeRepository) ReassignEmployees(fromDeptID, toDeptID int) error {
	result := r.db.Model(&domain.Employee{}).
		Where("department_id = ?", fromDeptID).
		Update("department_id", toDeptID)
	return result.Error
}