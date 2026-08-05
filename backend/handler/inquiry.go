package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/model"
	"github.com/AlanSyue/fenny-studio-booking/backend/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InquiryHandler struct {
	db       *pgxpool.Pool
	notifier *service.Notifier
}

func NewInquiryHandler(db *pgxpool.Pool, notifier *service.Notifier) *InquiryHandler {
	return &InquiryHandler{db: db, notifier: notifier}
}

type createInquiryRequest struct {
	Name              *string `json:"name"`
	Email             string  `json:"email"`
	CourseID          *int    `json:"course_id"`
	CourseIDs         []int   `json:"course_ids"`
	Headcount         *string `json:"headcount"`
	City              *string `json:"city"`
	District          *string `json:"district"`
	Address           *string `json:"address"`
	Budget            *string `json:"budget"`
	AdditionalContact *string `json:"additional_contact"`
}

// pgxQuerier is a minimal abstraction for *pgxpool.Pool or pgx.Tx.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Create handles public inquiry form submission
func (h *InquiryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		writeJSONError(w, http.StatusBadRequest, "Email is required")
		return
	}
	if req.City == nil || *req.City == "" {
		writeJSONError(w, http.StatusBadRequest, "City is required")
		return
	}
	if req.District == nil || *req.District == "" {
		writeJSONError(w, http.StatusBadRequest, "District is required")
		return
	}

	ctx := r.Context()
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(ctx)

	inquiryNo, err := service.NextInquiryNo(ctx, tx, time.Now())
	if err != nil {
		log.Printf("reserve inquiry number failed: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create inquiry")
		return
	}

	var inquiry model.Inquiry
	err = tx.QueryRow(ctx,
		`INSERT INTO inquiries (inquiry_no, name, email, course_id, headcount, city, district, address, budget, additional_contact)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, inquiry_no, name, email, course_id, headcount, city, district, address, budget, additional_contact, is_processed, created_at`,
		inquiryNo, req.Name, req.Email, req.CourseID, req.Headcount, req.City, req.District, req.Address, req.Budget, req.AdditionalContact,
	).Scan(&inquiry.ID, &inquiry.InquiryNo, &inquiry.Name, &inquiry.Email, &inquiry.CourseID, &inquiry.Headcount,
		&inquiry.City, &inquiry.District, &inquiry.Address, &inquiry.Budget,
		&inquiry.AdditionalContact, &inquiry.IsProcessed, &inquiry.CreatedAt)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create inquiry")
		return
	}

	// Insert inquiry_courses rows, deduplicated.
	if len(req.CourseIDs) > 0 {
		seen := make(map[int]bool, len(req.CourseIDs))
		batch := &pgx.Batch{}
		for _, cid := range req.CourseIDs {
			if seen[cid] {
				continue
			}
			seen[cid] = true
			batch.Queue(
				`INSERT INTO inquiry_courses (inquiry_id, course_id) VALUES ($1, $2)
				 ON CONFLICT DO NOTHING`,
				inquiry.ID, cid,
			)
		}
		br := tx.SendBatch(ctx, batch)
		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				br.Close()
				writeJSONError(w, http.StatusInternalServerError, "Failed to attach courses")
				return
			}
		}
		if err := br.Close(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to finalize courses")
			return
		}
	}

	// Populate inquiry.Courses from the join table.
	courses, err := fetchInquiryCourses(ctx, tx, inquiry.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load courses")
		return
	}
	inquiry.Courses = courses

	if err := tx.Commit(ctx); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to commit inquiry")
		return
	}

	// Notify admin via Telegram asynchronously. Failure must never affect the
	// 201 response, so it runs in its own goroutine and only logs on error. The
	// notifier uses context.Background() internally so it survives request end.
	go func(inq model.Inquiry) {
		if err := h.notifier.NotifyInquiry(inq); err != nil {
			log.Printf("telegram notify inquiry %d failed: %v", inq.ID, err)
		}
	}(inquiry)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inquiry)
}

// fetchInquiryCourses returns courses attached to a single inquiry.
func fetchInquiryCourses(ctx context.Context, q pgxQuerier, inquiryID int) ([]model.Course, error) {
	rows, err := q.Query(ctx,
		`SELECT c.id, c.name, c.image_path, c.description, c.product_url, c.woo_product_id, c.is_active, c.created_at, c.updated_at
		 FROM inquiry_courses ic
		 JOIN courses c ON c.id = ic.course_id
		 WHERE ic.inquiry_id = $1
		 ORDER BY c.id`,
		inquiryID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	courses := []model.Course{}
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Name, &c.ImagePath, &c.Description, &c.ProductURL, &c.WooProductID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

// List returns all inquiries with their courses joined (admin endpoint)
func (h *InquiryHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := h.db.Query(ctx,
		`SELECT i.id, i.inquiry_no, i.name, i.email, i.course_id, i.headcount, i.city, i.district, i.address, i.budget,
		        i.additional_contact, i.is_processed, i.created_at
		 FROM inquiries i
		 ORDER BY i.created_at DESC`,
	)
	if err != nil {
		http.Error(w, "Failed to fetch inquiries", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	inquiries := []model.Inquiry{}
	ids := []int{}
	for rows.Next() {
		var inq model.Inquiry
		if err := rows.Scan(&inq.ID, &inq.InquiryNo, &inq.Name, &inq.Email, &inq.CourseID, &inq.Headcount,
			&inq.City, &inq.District, &inq.Address, &inq.Budget,
			&inq.AdditionalContact, &inq.IsProcessed, &inq.CreatedAt); err != nil {
			http.Error(w, "Failed to scan inquiry", http.StatusInternalServerError)
			return
		}
		inq.Courses = []model.Course{}
		inquiries = append(inquiries, inq)
		ids = append(ids, inq.ID)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to iterate inquiries", http.StatusInternalServerError)
		return
	}

	if len(ids) > 0 {
		joinRows, err := h.db.Query(ctx,
			`SELECT ic.inquiry_id, c.id, c.name, c.image_path, c.description, c.product_url, c.woo_product_id, c.is_active, c.created_at, c.updated_at
			 FROM inquiry_courses ic
			 JOIN courses c ON c.id = ic.course_id
			 WHERE ic.inquiry_id = ANY($1)
			 ORDER BY ic.inquiry_id, c.id`,
			ids,
		)
		if err != nil {
			http.Error(w, "Failed to fetch inquiry courses", http.StatusInternalServerError)
			return
		}
		defer joinRows.Close()

		coursesByInquiry := map[int][]model.Course{}
		for joinRows.Next() {
			var inquiryID int
			var c model.Course
			if err := joinRows.Scan(&inquiryID, &c.ID, &c.Name, &c.ImagePath, &c.Description, &c.ProductURL, &c.WooProductID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
				http.Error(w, "Failed to scan inquiry course", http.StatusInternalServerError)
				return
			}
			coursesByInquiry[inquiryID] = append(coursesByInquiry[inquiryID], c)
		}
		if err := joinRows.Err(); err != nil {
			http.Error(w, "Failed to iterate inquiry courses", http.StatusInternalServerError)
			return
		}
		for i := range inquiries {
			if cs, ok := coursesByInquiry[inquiries[i].ID]; ok {
				inquiries[i].Courses = cs
			}
		}
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
		 RETURNING id, inquiry_no, name, email, course_id, headcount, city, district, address, budget, additional_contact, is_processed, created_at`,
		id,
	).Scan(&inquiry.ID, &inquiry.InquiryNo, &inquiry.Name, &inquiry.Email, &inquiry.CourseID, &inquiry.Headcount,
		&inquiry.City, &inquiry.District, &inquiry.Address, &inquiry.Budget,
		&inquiry.AdditionalContact, &inquiry.IsProcessed, &inquiry.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to update inquiry", http.StatusInternalServerError)
		return
	}

	// Attach courses for consistency with List.
	courses, err := fetchInquiryCourses(r.Context(), h.db, inquiry.ID)
	if err == nil {
		inquiry.Courses = courses
	} else {
		inquiry.Courses = []model.Course{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inquiry)
}
