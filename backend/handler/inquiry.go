package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/AlanSyue/fenny-studio-booking/backend/model"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InquiryHandler struct {
	db *pgxpool.Pool
}

func NewInquiryHandler(db *pgxpool.Pool) *InquiryHandler {
	return &InquiryHandler{db: db}
}

type createInquiryRequest struct {
	Name              *string `json:"name"`
	Email             string  `json:"email"`
	CourseID          *int    `json:"course_id"`
	Headcount         *string `json:"headcount"`
	AdditionalContact *string `json:"additional_contact"`
}

// Create handles public inquiry form submission
func (h *InquiryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	var inquiry model.Inquiry
	err := h.db.QueryRow(r.Context(),
		`INSERT INTO inquiries (name, email, course_id, headcount, additional_contact)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, email, course_id, headcount, additional_contact, is_processed, created_at`,
		req.Name, req.Email, req.CourseID, req.Headcount, req.AdditionalContact,
	).Scan(&inquiry.ID, &inquiry.Name, &inquiry.Email, &inquiry.CourseID, &inquiry.Headcount, &inquiry.AdditionalContact, &inquiry.IsProcessed, &inquiry.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to create inquiry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inquiry)
}

// List returns all inquiries with course name joined (admin endpoint)
func (h *InquiryHandler) List(w http.ResponseWriter, r *http.Request) {
	query := `SELECT i.id, i.name, i.email, i.course_id, c.name as course_name, i.headcount, i.additional_contact, i.is_processed, i.created_at
		FROM inquiries i
		LEFT JOIN courses c ON i.course_id = c.id
		ORDER BY i.created_at DESC`

	rows, err := h.db.Query(r.Context(), query)
	if err != nil {
		http.Error(w, "Failed to fetch inquiries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var inquiries []model.Inquiry
	for rows.Next() {
		var inq model.Inquiry
		if err := rows.Scan(&inq.ID, &inq.Name, &inq.Email, &inq.CourseID, &inq.CourseName, &inq.Headcount, &inq.AdditionalContact, &inq.IsProcessed, &inq.CreatedAt); err != nil {
			http.Error(w, "Failed to scan inquiry", http.StatusInternalServerError)
			return
		}
		inquiries = append(inquiries, inq)
	}

	if inquiries == nil {
		inquiries = []model.Inquiry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inquiries)
}

// ToggleProcessed toggles the is_processed status of an inquiry
func (h *InquiryHandler) ToggleProcessed(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid inquiry ID", http.StatusBadRequest)
		return
	}

	var inquiry model.Inquiry
	err = h.db.QueryRow(r.Context(),
		`UPDATE inquiries SET is_processed = NOT is_processed WHERE id = $1
		 RETURNING id, name, email, course_id, headcount, additional_contact, is_processed, created_at`,
		id,
	).Scan(&inquiry.ID, &inquiry.Name, &inquiry.Email, &inquiry.CourseID, &inquiry.Headcount, &inquiry.AdditionalContact, &inquiry.IsProcessed, &inquiry.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to update inquiry", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inquiry)
}
