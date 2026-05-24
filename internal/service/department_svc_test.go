package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merclamp/org-api/internal/domain"
)

type mockDepartmentRepo struct {
	departments map[int]*domain.Department
	nextID      int
}

func newMockDeptRepo() *mockDepartmentRepo {
	return &mockDepartmentRepo{
		departments: make(map[int]*domain.Department),
		nextID:      1,
	}
}

func (m *mockDepartmentRepo) Create(dept *domain.Department) error {
	dept.ID = m.nextID
	m.nextID++
	m.departments[dept.ID] = dept
	return nil
}

func (m *mockDepartmentRepo) FindByID(id int) (*domain.Department, error) {
	dept, ok := m.departments[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	copy := *dept
	return &copy, nil
}

func (m *mockDepartmentRepo) FindChildren(parentID int) ([]domain.Department, error) {
	var children []domain.Department
	for _, dept := range m.departments {
		if dept.ParentID != nil && *dept.ParentID == parentID {
			children = append(children, *dept)
		}
	}
	return children, nil
}

func (m *mockDepartmentRepo) Update(dept *domain.Department) error {
	if _, ok := m.departments[dept.ID]; !ok {
		return domain.ErrNotFound
	}
	m.departments[dept.ID] = dept
	return nil
}

func (m *mockDepartmentRepo) Delete(id int) error {
	if _, ok := m.departments[id]; !ok {
		return domain.ErrNotFound
	}
	for _, dept := range m.departments {
		if dept.ParentID != nil && *dept.ParentID == id {
			m.Delete(dept.ID)
		}
	}
	delete(m.departments, id)
	return nil
}

func (m *mockDepartmentRepo) ExistsByNameAndParent(name string, parentID *int, excludeID int) (bool, error) {
	for _, dept := range m.departments {
		if dept.ID == excludeID {
			continue
		}
		if dept.Name != name {
			continue
		}
		// Сравниваем parent_id
		if parentID == nil && dept.ParentID == nil {
			return true, nil
		}
		if parentID != nil && dept.ParentID != nil && *parentID == *dept.ParentID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockDepartmentRepo) GetAncestorIDs(id int) ([]int, error) {
	var ancestors []int
	currentID := id
	visited := make(map[int]bool)

	for {
		dept, ok := m.departments[currentID]
		if !ok || dept.ParentID == nil {
			break
		}
		if visited[*dept.ParentID] {
			break
		}
		ancestors = append(ancestors, *dept.ParentID)
		visited[*dept.ParentID] = true
		currentID = *dept.ParentID
	}
	return ancestors, nil
}

type mockEmployeeRepo struct {
	employees map[int]*domain.Employee
	nextID    int
}

func newMockEmpRepo() *mockEmployeeRepo {
	return &mockEmployeeRepo{
		employees: make(map[int]*domain.Employee),
		nextID:    1,
	}
}

func (m *mockEmployeeRepo) Create(emp *domain.Employee) error {
	emp.ID = m.nextID
	m.nextID++
	m.employees[emp.ID] = emp
	return nil
}

func (m *mockEmployeeRepo) FindByDepartmentID(deptID int) ([]domain.Employee, error) {
	var result []domain.Employee
	for _, emp := range m.employees {
		if emp.DepartmentID == deptID {
			result = append(result, *emp)
		}
	}
	return result, nil
}

func (m *mockEmployeeRepo) ReassignEmployees(fromDeptID, toDeptID int) error {
	for _, emp := range m.employees {
		if emp.DepartmentID == fromDeptID {
			emp.DepartmentID = toDeptID
		}
	}
	return nil
}


func setupDeptService() (*DepartmentService, *mockDepartmentRepo, *mockEmployeeRepo) {
	deptRepo := newMockDeptRepo()
	empRepo := newMockEmpRepo()
	svc := NewDepartmentService(deptRepo, empRepo)
	return svc, deptRepo, empRepo
}

func intPtr(v int) *int {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func TestCreateDepartment_Success(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, err := svc.Create(CreateDepartmentInput{
		Name: "Engineering",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, dept.ID)
	assert.Equal(t, "Engineering", dept.Name)
	assert.Nil(t, dept.ParentID)
}

func TestCreateDepartment_WithParent(t *testing.T) {
	svc, _, _ := setupDeptService()

	root, err := svc.Create(CreateDepartmentInput{Name: "Company"})
	require.NoError(t, err)

	child, err := svc.Create(CreateDepartmentInput{
		Name:     "Backend",
		ParentID: &root.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, child.ID)
	assert.Equal(t, "Backend", child.Name)
	assert.Equal(t, root.ID, *child.ParentID)
}

func TestCreateDepartment_EmptyName(t *testing.T) {
	svc, _, _ := setupDeptService()

	_, err := svc.Create(CreateDepartmentInput{Name: ""})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateDepartment_WhitespaceName(t *testing.T) {
	svc, _, _ := setupDeptService()

	_, err := svc.Create(CreateDepartmentInput{Name: "   "})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestCreateDepartment_TrimsName(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, err := svc.Create(CreateDepartmentInput{Name: "  Backend  "})

	require.NoError(t, err)
	assert.Equal(t, "Backend", dept.Name)
}

func TestCreateDepartment_NonExistentParent(t *testing.T) {
	svc, _, _ := setupDeptService()

	_, err := svc.Create(CreateDepartmentInput{
		Name:     "Backend",
		ParentID: intPtr(999),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestCreateDepartment_DuplicateName(t *testing.T) {
	svc, _, _ := setupDeptService()

	_, err := svc.Create(CreateDepartmentInput{Name: "Backend"})
	require.NoError(t, err)

	_, err = svc.Create(CreateDepartmentInput{Name: "Backend"})
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrConflict)
}

func TestCreateDepartment_SameNameDifferentParent(t *testing.T) {
	svc, _, _ := setupDeptService()

	root1, err := svc.Create(CreateDepartmentInput{Name: "Division A"})
	require.NoError(t, err)

	root2, err := svc.Create(CreateDepartmentInput{Name: "Division B"})
	require.NoError(t, err)

	_, err = svc.Create(CreateDepartmentInput{Name: "Backend", ParentID: &root1.ID})
	require.NoError(t, err)

	// То же имя но в другом parent — должно быть OK
	_, err = svc.Create(CreateDepartmentInput{Name: "Backend", ParentID: &root2.ID})
	require.NoError(t, err)
}

func TestGetDepartment_NotFound(t *testing.T) {
	svc, _, _ := setupDeptService()

	_, err := svc.Get(999, GetDepartmentInput{Depth: 1, IncludeEmployees: true})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetDepartment_WithEmployees(t *testing.T) {
	svc, _, empRepo := setupDeptService()

	dept, err := svc.Create(CreateDepartmentInput{Name: "Backend"})
	require.NoError(t, err)

	empRepo.Create(&domain.Employee{
		DepartmentID: dept.ID,
		FullName:     "Ivan Ivanov",
		Position:     "Developer",
	})

	result, err := svc.Get(dept.ID, GetDepartmentInput{
		Depth:            1,
		IncludeEmployees: true,
	})

	require.NoError(t, err)
	assert.Len(t, result.Employees, 1)
	assert.Equal(t, "Ivan Ivanov", result.Employees[0].FullName)
}

func TestGetDepartment_WithoutEmployees(t *testing.T) {
	svc, _, empRepo := setupDeptService()

	dept, err := svc.Create(CreateDepartmentInput{Name: "Backend"})
	require.NoError(t, err)

	empRepo.Create(&domain.Employee{
		DepartmentID: dept.ID,
		FullName:     "Ivan",
		Position:     "Dev",
	})

	result, err := svc.Get(dept.ID, GetDepartmentInput{
		Depth:            1,
		IncludeEmployees: false,
	})

	require.NoError(t, err)
	assert.Nil(t, result.Employees)
}

func TestGetDepartment_DepthClamp(t *testing.T) {
	svc, _, _ := setupDeptService()

	root, _ := svc.Create(CreateDepartmentInput{Name: "Root"})
	child, _ := svc.Create(CreateDepartmentInput{Name: "Child", ParentID: &root.ID})
	_, _ = svc.Create(CreateDepartmentInput{Name: "Grandchild", ParentID: &child.ID})

	result, err := svc.Get(root.ID, GetDepartmentInput{Depth: 1, IncludeEmployees: false})
	require.NoError(t, err)
	assert.Len(t, result.Children, 1)
	assert.Len(t, result.Children[0].Children, 0) // внуков нет при depth=1

	result, err = svc.Get(root.ID, GetDepartmentInput{Depth: 2, IncludeEmployees: false})
	require.NoError(t, err)
	assert.Len(t, result.Children, 1)
	assert.Len(t, result.Children[0].Children, 1)
}

func TestUpdateDepartment_Rename(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, _ := svc.Create(CreateDepartmentInput{Name: "Old Name"})

	updated, err := svc.Update(dept.ID, UpdateDepartmentInput{
		Name: strPtr("New Name"),
	})

	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestUpdateDepartment_MoveToAnotherParent(t *testing.T) {
	svc, _, _ := setupDeptService()

	root1, _ := svc.Create(CreateDepartmentInput{Name: "Root 1"})
	root2, _ := svc.Create(CreateDepartmentInput{Name: "Root 2"})
	child, _ := svc.Create(CreateDepartmentInput{Name: "Child", ParentID: &root1.ID})

	updated, err := svc.Update(child.ID, UpdateDepartmentInput{
		ParentID: &root2.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, root2.ID, *updated.ParentID)
}

func TestUpdateDepartment_MakeRoot(t *testing.T) {
	svc, _, _ := setupDeptService()

	root, _ := svc.Create(CreateDepartmentInput{Name: "Root"})
	child, _ := svc.Create(CreateDepartmentInput{Name: "Child", ParentID: &root.ID})

	updated, err := svc.Update(child.ID, UpdateDepartmentInput{
		ClearParent: true,
	})

	require.NoError(t, err)
	assert.Nil(t, updated.ParentID)
}

func TestUpdateDepartment_SelfReference(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, _ := svc.Create(CreateDepartmentInput{Name: "Dept"})

	_, err := svc.Update(dept.ID, UpdateDepartmentInput{
		ParentID: &dept.ID,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCyclicReference)
}

func TestUpdateDepartment_CyclicReference(t *testing.T) {
	svc, _, _ := setupDeptService()

	root, _ := svc.Create(CreateDepartmentInput{Name: "Root"})
	child, _ := svc.Create(CreateDepartmentInput{Name: "Child", ParentID: &root.ID})
	grandchild, _ := svc.Create(CreateDepartmentInput{Name: "Grandchild", ParentID: &child.ID})

	_, err := svc.Update(root.ID, UpdateDepartmentInput{
		ParentID: &grandchild.ID,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrCyclicReference)
}

func TestUpdateDepartment_NotFound(t *testing.T) {
	svc, _, _ := setupDeptService()

	_, err := svc.Update(999, UpdateDepartmentInput{
		Name: strPtr("Nope"),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateDepartment_DuplicateName(t *testing.T) {
	svc, _, _ := setupDeptService()

	root, _ := svc.Create(CreateDepartmentInput{Name: "Root"})
	_, _ = svc.Create(CreateDepartmentInput{Name: "Backend", ParentID: &root.ID})
	child2, _ := svc.Create(CreateDepartmentInput{Name: "Frontend", ParentID: &root.ID})

	// Переименовываем Frontend в Backend внутри того же parent → конфликт
	_, err := svc.Update(child2.ID, UpdateDepartmentInput{
		Name: strPtr("Backend"),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrConflict)
}


func TestDeleteDepartment_Cascade(t *testing.T) {
	svc, deptRepo, empRepo := setupDeptService()

	root, _ := svc.Create(CreateDepartmentInput{Name: "Root"})
	child, _ := svc.Create(CreateDepartmentInput{Name: "Child", ParentID: &root.ID})

	empRepo.Create(&domain.Employee{DepartmentID: child.ID, FullName: "Test", Position: "Dev"})

	err := svc.Delete(root.ID, DeleteDepartmentInput{Mode: DeleteModeCascade})

	require.NoError(t, err)
	assert.Empty(t, deptRepo.departments)
}

func TestDeleteDepartment_Reassign(t *testing.T) {
	svc, deptRepo, empRepo := setupDeptService()

	dept1, _ := svc.Create(CreateDepartmentInput{Name: "Dept 1"})
	dept2, _ := svc.Create(CreateDepartmentInput{Name: "Dept 2"})

	empRepo.Create(&domain.Employee{DepartmentID: dept1.ID, FullName: "Ivan", Position: "Dev"})

	err := svc.Delete(dept1.ID, DeleteDepartmentInput{
		Mode:                   DeleteModeReassign,
		ReassignToDepartmentID: &dept2.ID,
	})

	require.NoError(t, err)

	// Подразделение удалено
	_, exists := deptRepo.departments[dept1.ID]
	assert.False(t, exists)

	// Сотрудник переведён
	for _, emp := range empRepo.employees {
		assert.Equal(t, dept2.ID, emp.DepartmentID)
	}
}

func TestDeleteDepartment_ReassignWithoutTarget(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, _ := svc.Create(CreateDepartmentInput{Name: "Dept"})

	err := svc.Delete(dept.ID, DeleteDepartmentInput{
		Mode: DeleteModeReassign,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestDeleteDepartment_ReassignToSelf(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, _ := svc.Create(CreateDepartmentInput{Name: "Dept"})

	err := svc.Delete(dept.ID, DeleteDepartmentInput{
		Mode:                   DeleteModeReassign,
		ReassignToDepartmentID: &dept.ID,
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}

func TestDeleteDepartment_ReassignToNonExistent(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, _ := svc.Create(CreateDepartmentInput{Name: "Dept"})

	err := svc.Delete(dept.ID, DeleteDepartmentInput{
		Mode:                   DeleteModeReassign,
		ReassignToDepartmentID: intPtr(999),
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteDepartment_NotFound(t *testing.T) {
	svc, _, _ := setupDeptService()

	err := svc.Delete(999, DeleteDepartmentInput{Mode: DeleteModeCascade})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteDepartment_InvalidMode(t *testing.T) {
	svc, _, _ := setupDeptService()

	dept, _ := svc.Create(CreateDepartmentInput{Name: "Dept"})

	err := svc.Delete(dept.ID, DeleteDepartmentInput{Mode: "invalid"})

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrValidation)
}