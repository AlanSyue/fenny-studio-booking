package model

import "time"

type Inquiry struct {
	ID                int       `json:"id"`
	Name              *string   `json:"name"`
	Email             string    `json:"email"`
	CourseID          *int      `json:"course_id"`
	CourseName        *string   `json:"course_name,omitempty"`
	Headcount         *string   `json:"headcount"`
	AdditionalContact *string   `json:"additional_contact"`
	IsProcessed       bool      `json:"is_processed"`
	CreatedAt         time.Time `json:"created_at"`
}
