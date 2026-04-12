package model

import "time"

type Inquiry struct {
	ID                int       `json:"id"`
	Name              *string   `json:"name"`
	Email             string    `json:"email"`
	CourseID          *int      `json:"course_id"`
	CourseName        *string   `json:"course_name,omitempty"`
	Courses           []Course  `json:"courses"`
	Headcount         *string   `json:"headcount"`
	City              *string   `json:"city"`
	District          *string   `json:"district"`
	Address           *string   `json:"address"`
	Budget            *string   `json:"budget"`
	AdditionalContact *string   `json:"additional_contact"`
	IsProcessed       bool      `json:"is_processed"`
	CreatedAt         time.Time `json:"created_at"`
}
