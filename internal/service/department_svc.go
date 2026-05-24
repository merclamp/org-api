package service

import (
	"fmt"
	"strings"

	"github.com/merclamp/org-api/internal/domain"
)

type DepartmentRepo interface {
	Create(dept *domain.Department) error
	FindByID(id int) (*domain.Department, error)
	FindChildren(parentID int) ([]domain.Department, error)
	Update(dept *domain.Department) error
	Delete(id int) error
	ExistsByNameAndParent(name string, parentID *int, excludeID int) (bool, error)
	GetAncestorIDs(id int) ([]int, error)
}

type EmployeeRepo interface {
	Create(emp *domain.Employee) error
	FindByDepartmentID(departmentID int) ([]domain.Employee, error)
	ReassignEmployees(fromDeptID, toDeptID int) error
}

type CreateDepartmentInput struct {
	Name     string
	ParentID *int
}

type UpdateDepartmentInput struct {
	Name     *string // nil = не менять
	ParentID *int    // nil = не менять; &0 = сделать root (обнулить parent)
	ClearParent bool  // true = явно передали parent_id: null
}

type DeleteDepartmentInput struct {
	Mode                  string // "cascade" | "reassign"
	ReassignToDepartmentID *int
}

type GetDepartmentInput struct {
	Depth            int
	IncludeEmployees bool
}

const (
	DeleteModeCascade  = "cascade"
	DeleteModeReassign = "reassign"

	MaxDepth     = 5
	DefaultDepth = 1

	MaxNameLength = 200
)

type DepartmentService struct {
	deptRepo DepartmentRepo
	empRepo  EmployeeRepo
}

func NewDepartmentService(deptRepo DepartmentRepo, empRepo EmployeeRepo) *DepartmentService {
	return &DepartmentService{
		deptRepo: deptRepo,
		empRepo:  empRepo,
	}
}

func (s *DepartmentService) Create(input CreateDepartmentInput) (*domain.Department, error) {
	name, err := validateName(input.Name)
	if err != nil {
		return nil, err
	}

	if input.ParentID != nil {
		if _, err := s.deptRepo.FindByID(*input.ParentID); err != nil {
			if err == domain.ErrNotFound {
				return nil, fmt.Errorf("%w: parent department %d not found", domain.ErrNotFound, *input.ParentID)
			}
			return nil, err
		}
	}

	exists, err := s.deptRepo.ExistsByNameAndParent(name, input.ParentID, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: department with name %q already exists in this parent", domain.ErrConflict, name)
	}

	dept := &domain.Department{
		Name:     name,
		ParentID: input.ParentID,
	}

	if err := s.deptRepo.Create(dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *DepartmentService) Get(id int, input GetDepartmentInput) (*domain.Department, error) {
	depth := input.Depth
	if depth < 1 {
		depth = DefaultDepth
	}
	if depth > MaxDepth {
		depth = MaxDepth
	}

	dept, err := s.deptRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if input.IncludeEmployees {
		employees, err := s.empRepo.FindByDepartmentID(dept.ID)
		if err != nil {
			return nil, err
		}
		dept.Employees = employees
	}

	if err := s.buildTree(dept, 1, depth, input.IncludeEmployees); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *DepartmentService) buildTree(dept *domain.Department, currentDepth, maxDepth int, includeEmployees bool) error {
	if currentDepth >= maxDepth {
		return nil
	}

	children, err := s.deptRepo.FindChildren(dept.ID)
	if err != nil {
		return err
	}

	if len(children) == 0 {
		return nil
	}

	for i := range children {
		child := &children[i]

		if includeEmployees {
			employees, err := s.empRepo.FindByDepartmentID(child.ID)
			if err != nil {
				return err
			}
			child.Employees = employees
		}

		if err := s.buildTree(child, currentDepth+1, maxDepth, includeEmployees); err != nil {
			return err
		}
	}

	dept.Children = children
	return nil
}

func (s *DepartmentService) Update(id int, input UpdateDepartmentInput) (*domain.Department, error) {
	dept, err := s.deptRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name, err := validateName(*input.Name)
		if err != nil {
			return nil, err
		}
		dept.Name = name
	}

	newParentID := dept.ParentID

	if input.ClearParent {
		newParentID = nil
	} else if input.ParentID != nil {
		newParentID = input.ParentID
	}

	if err := s.validateParentChange(dept, newParentID); err != nil {
		return nil, err
	}

	dept.ParentID = newParentID

	exists, err := s.deptRepo.ExistsByNameAndParent(dept.Name, dept.ParentID, dept.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: department with name %q already exists in this parent", domain.ErrConflict, dept.Name)
	}

	if err := s.deptRepo.Update(dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *DepartmentService) validateParentChange(dept *domain.Department, newParentID *int) error {
	if newParentID == nil {
		return nil
	}

	if *newParentID == dept.ID {
		return fmt.Errorf("%w: department cannot be its own parent", domain.ErrCyclicReference)
	}

	if _, err := s.deptRepo.FindByID(*newParentID); err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("%w: parent department %d not found", domain.ErrNotFound, *newParentID)
		}
		return err
	}

	cycle, err := s.wouldCreateCycle(dept.ID, *newParentID)
	if err != nil {
		return err
	}
	if cycle {
		return fmt.Errorf("%w: cannot move department into its own subtree", domain.ErrCyclicReference)
	}

	return nil
}

func (s *DepartmentService) wouldCreateCycle(departmentID, candidateParentID int) (bool, error) {
	ancestors, err := s.deptRepo.GetAncestorIDs(candidateParentID)
	if err != nil {
		return false, err
	}

	for _, ancestorID := range ancestors {
		if ancestorID == departmentID {
			return true, nil
		}
	}

	isDescendant, err := s.isDescendant(departmentID, candidateParentID)
	if err != nil {
		return false, err
	}

	return isDescendant, nil
}

func (s *DepartmentService) isDescendant(rootID, targetID int) (bool, error) {
	children, err := s.deptRepo.FindChildren(rootID)
	if err != nil {
		return false, err
	}

	for _, child := range children {
		if child.ID == targetID {
			return true, nil
		}
		found, err := s.isDescendant(child.ID, targetID)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}

	return false, nil
}

func (s *DepartmentService) Delete(id int, input DeleteDepartmentInput) error {
	// Проверяем что подразделение существует
	if _, err := s.deptRepo.FindByID(id); err != nil {
		return err
	}

	switch input.Mode {
	case DeleteModeCascade:
		return s.deleteCascade(id)

	case DeleteModeReassign:
		return s.deleteReassign(id, input.ReassignToDepartmentID)

	default:
		return fmt.Errorf("%w: unknown delete mode %q, use 'cascade' or 'reassign'", domain.ErrValidation, input.Mode)
	}
}

func (s *DepartmentService) deleteCascade(id int) error {
	return s.deptRepo.Delete(id)
}

func (s *DepartmentService) deleteReassign(id int, reassignTo *int) error {
	if reassignTo == nil {
		return fmt.Errorf("%w: reassign_to_department_id is required for reassign mode", domain.ErrValidation)
	}

	if _, err := s.deptRepo.FindByID(*reassignTo); err != nil {
		if err == domain.ErrNotFound {
			return fmt.Errorf("%w: reassign target department %d not found", domain.ErrNotFound, *reassignTo)
		}
		return err
	}

	if *reassignTo == id {
		return fmt.Errorf("%w: cannot reassign employees to the department being deleted", domain.ErrValidation)
	}

	if err := s.empRepo.ReassignEmployees(id, *reassignTo); err != nil {
		return err
	}

	return s.deptRepo.Delete(id)
}

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: name cannot be empty", domain.ErrValidation)
	}
	if len(name) > MaxNameLength {
		return "", fmt.Errorf("%w: name cannot exceed %d characters", domain.ErrValidation, MaxNameLength)
	}
	return name, nil
}