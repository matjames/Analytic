package models

import "time"

type AdminLevel struct {
	ID          int64     `json:"id"`
	MflUid      string    `json:"mfl_uid" db:"mfl_uid"`
	Name        string    `json:"name"`
	LevelNumber int       `json:"level_number" db:"level_number"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

