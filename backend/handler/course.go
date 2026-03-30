package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
	"github.com/AlanSyue/fenny-studio-booking/backend/model"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseHandler struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewCourseHandler(db *pgxpool.Pool, cfg *config.Config) *CourseHandler {
	return &CourseHandler{db: db, cfg: cfg}
}

// ListActive returns active courses (public endpoint for the form)
func (h *CourseHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(),
		"SELECT id, name, image_path, description, is_active, created_at, updated_at FROM courses WHERE is_active = true ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []model.Course
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Name, &c.ImagePath, &c.Description, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan course", http.StatusInternalServerError)
			return
		}
		courses = append(courses, c)
	}

	if courses == nil {
		courses = []model.Course{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(courses)
}

// ListAll returns all courses including inactive (admin endpoint)
func (h *CourseHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(),
		"SELECT id, name, image_path, description, is_active, created_at, updated_at FROM courses ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []model.Course
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Name, &c.ImagePath, &c.Description, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan course", http.StatusInternalServerError)
			return
		}
		courses = append(courses, c)
	}

	if courses == nil {
		courses = []model.Course{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(courses)
}

// Create handles course creation with image upload (multipart form)
func (h *CourseHandler) Create(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	var description *string
	if desc := r.FormValue("description"); desc != "" {
		description = &desc
	}

	var imagePath *string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename, err := saveUploadedFile(file, header, h.cfg.UploadDir)
		if err != nil {
			http.Error(w, "Failed to upload image", http.StatusInternalServerError)
			return
		}
		imagePath = &filename
	}

	if imagePath == nil {
		if p := r.FormValue("image_path"); p != "" {
			imagePath = &p
		}
	}

	var course model.Course
	err = h.db.QueryRow(r.Context(),
		"INSERT INTO courses (name, image_path, description) VALUES ($1, $2, $3) RETURNING id, name, image_path, description, is_active, created_at, updated_at",
		name, imagePath, description,
	).Scan(&course.ID, &course.Name, &course.ImagePath, &course.Description, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
	if err != nil {
		http.Error(w, "Failed to create course", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(course)
}

// Update handles course update with optional image upload
func (h *CourseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	var description *string
	if desc := r.FormValue("description"); desc != "" {
		description = &desc
	}

	// Check if a new image was uploaded
	var imagePath *string
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename, err := saveUploadedFile(file, header, h.cfg.UploadDir)
		if err != nil {
			http.Error(w, "Failed to upload image", http.StatusInternalServerError)
			return
		}
		imagePath = &filename
	}

	if imagePath == nil {
		if p := r.FormValue("image_path"); p != "" {
			imagePath = &p
		}
	}

	var course model.Course
	if imagePath != nil {
		err = h.db.QueryRow(r.Context(),
			"UPDATE courses SET name = $1, image_path = $2, description = $3, updated_at = NOW() WHERE id = $4 RETURNING id, name, image_path, description, is_active, created_at, updated_at",
			name, imagePath, description, id,
		).Scan(&course.ID, &course.Name, &course.ImagePath, &course.Description, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
	} else {
		err = h.db.QueryRow(r.Context(),
			"UPDATE courses SET name = $1, description = $2, updated_at = NOW() WHERE id = $3 RETURNING id, name, image_path, description, is_active, created_at, updated_at",
			name, description, id,
		).Scan(&course.ID, &course.Name, &course.ImagePath, &course.Description, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
	}
	if err != nil {
		http.Error(w, "Failed to update course", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(course)
}

// Delete soft-deletes a course
func (h *CourseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(r.Context(),
		"UPDATE courses SET is_active = false, updated_at = NOW() WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete course", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
