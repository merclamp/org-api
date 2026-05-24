package domain

import "time"

type Employee struct {
	ID           int        `gorm:"primaryKey" json:"id"`
	DepartmentID int        `gorm:"not null"   json:"department_id"`
	FullName     string     `gorm:"not null"   json:"full_name"`
	Position     string     `gorm:"not null"   json:"position"`
	HiredAt      *time.Time `                  json:"hired_at"`
	CreatedAt    time.Time  `                  json:"created_at"`
}