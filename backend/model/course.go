package model

import "time"

type Course struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	ImagePath   *string   `json:"image_path"`
	Description *string   `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
