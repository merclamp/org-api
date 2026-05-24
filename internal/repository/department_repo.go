package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/merclamp/org-api/internal/domain"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

func (r *DepartmentRepository) Create(dept *domain.Department) error {
	result := r.db.Create(dept)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return domain.ErrConflict
		}
		return result.Error
	}
	return nil
}

func (r *DepartmentRepository) FindByID(id int) (*domain.Department, error) {
	var dept domain.Department
	result := r.db.First(&dept, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, result.Error
	}
	return &dept, nil
}

func (r *DepartmentRepository) FindChildren(parentID int) ([]domain.Department, error) {
	var children []domain.Department
	result := r.db.Where("parent_id = ?", parentID).
		Order("name ASC").
		Find(&children)
	if result.Error != nil {
		return nil, result.Error
	}
	return children, nil
}

func (r *DepartmentRepository) Update(dept *domain.Department) error {
	result := r.db.Save(dept)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return domain.ErrConflict
		}
		return result.Error
	}
	return nil
}

func (r *DepartmentRepository) Delete(id int) error {
	result := r.db.Delete(&domain.Department{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DepartmentRepository) ExistsByNameAndParent(name string, parentID *int, excludeID int) (bool, error) {
	query := r.db.Model(&domain.Department{}).Where("LOWER(name) = LOWER(?)", name)

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *DepartmentRepository) GetAncestorIDs(id int) ([]int, error) {
	var ancestors []int
	currentID := id

	visited := make(map[int]bool)

	for {
		var dept domain.Department
		result := r.db.Select("id, parent_id").First(&dept, currentID)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				break
			}
			return nil, result.Error
		}

		if dept.ParentID == nil {
			break
		}

		if visited[*dept.ParentID] {
			break // защита от бесконечного цикла в испорченных данных
		}

		ancestors = append(ancestors, *dept.ParentID)
		visited[*dept.ParentID] = true
		currentID = *dept.ParentID
	}

	return ancestors, nil
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "23505") ||
		strings.Contains(err.Error(), "duplicate key")
}