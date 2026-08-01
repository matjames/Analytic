package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	FirstName    *string   `json:"first_name,omitempty" db:"first_name"`
	LastName     *string   `json:"last_name,omitempty" db:"last_name"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Role         *string   `json:"role"`
	Password     string    `json:"password,omitempty"`
	PasswordHash string    `json:"-" db:"password"`
	Organisation *string   `json:"organisation"`
	Phoneno      *string   `json:"phoneno"`
	DistrictID         *string   `json:"district_id" db:"district_id"`
	MustChangePassword bool      `json:"must_change_password" db:"must_change_password"`
	CreatedAt          time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt" db:"updatedAt"`
}
