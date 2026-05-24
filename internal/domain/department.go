package domain

import "time"

type Department struct {
	ID        int          `gorm:"primaryKey"        json:"id"`
	Name      string       `gorm:"not null"          json:"name"`
	ParentID  *int         `gorm:"index"             json:"parent_id"`
	CreatedAt time.Time    `                         json:"created_at"`

	Children  []Department `gorm:"-" json:"children,omitempty"`
	Employees []Employee   `gorm:"-" json:"employees,omitempty"`
}