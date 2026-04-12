package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
	"github.com/AlanSyue/fenny-studio-booking/backend/model"
	"github.com/AlanSyue/fenny-studio-booking/backend/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CourseHandler struct {
	db        *pgxpool.Pool
	cfg       *config.Config
	wooClient *service.Client
}

func NewCourseHandler(db *pgxpool.Pool, cfg *config.Config, wooClient *service.Client) *CourseHandler {
	return &CourseHandler{db: db, cfg: cfg, wooClient: wooClient}
}

// ListActive returns active courses (public endpoint for the form)
func (h *CourseHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(),
		"SELECT id, name, image_path, description, product_url, woo_product_id, is_active, created_at, updated_at FROM courses WHERE is_active = true ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []model.Course
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Name, &c.ImagePath, &c.Description, &c.ProductURL, &c.WooProductID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
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
		"SELECT id, name, image_path, description, product_url, woo_product_id, is_active, created_at, updated_at FROM courses ORDER BY is_active DESC, created_at DESC")
	if err != nil {
		http.Error(w, "Failed to fetch courses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var courses []model.Course
	for rows.Next() {
		var c model.Course
		if err := rows.Scan(&c.ID, &c.Name, &c.ImagePath, &c.Description, &c.ProductURL, &c.WooProductID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
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

	var productURL *string
	if p := r.FormValue("product_url"); p != "" {
		productURL = &p
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
		`INSERT INTO courses (name, image_path, description, product_url)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, image_path, description, product_url, woo_product_id, is_active, created_at, updated_at`,
		name, imagePath, description, productURL,
	).Scan(&course.ID, &course.Name, &course.ImagePath, &course.Description, &course.ProductURL, &course.WooProductID, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
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

	var productURL *string
	if p := r.FormValue("product_url"); p != "" {
		productURL = &p
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
			`UPDATE courses SET name = $1, image_path = $2, description = $3, product_url = $4, updated_at = NOW()
			 WHERE id = $5
			 RETURNING id, name, image_path, description, product_url, woo_product_id, is_active, created_at, updated_at`,
			name, imagePath, description, productURL, id,
		).Scan(&course.ID, &course.Name, &course.ImagePath, &course.Description, &course.ProductURL, &course.WooProductID, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
	} else {
		err = h.db.QueryRow(r.Context(),
			`UPDATE courses SET name = $1, description = $2, product_url = $3, updated_at = NOW()
			 WHERE id = $4
			 RETURNING id, name, image_path, description, product_url, woo_product_id, is_active, created_at, updated_at`,
			name, description, productURL, id,
		).Scan(&course.ID, &course.Name, &course.ImagePath, &course.Description, &course.ProductURL, &course.WooProductID, &course.IsActive, &course.CreatedAt, &course.UpdatedAt)
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

// SyncFromWoo pulls products from WooCommerce and upserts them as courses.
func (h *CourseHandler) SyncFromWoo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h.cfg.WooAPIKey == "" || h.cfg.WooAPISecret == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "WooCommerce credentials not configured"})
		return
	}

	category, err := h.wooClient.GetCategoryBySlug(r.Context(), h.cfg.WooCategorySlug)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to resolve category: %v", err)})
		return
	}

	products, err := h.wooClient.ListProductsByCategory(r.Context(), category.ID)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to fetch products: %v", err)})
		return
	}

	// Whitelist：只保留名稱含 "材料包" 的商品；其餘一律略過，讓後續 deactivation 步驟自動把既有的停用掉。
	skipped := 0
	filtered := products[:0]
	for _, p := range products {
		if !strings.Contains(p.Name, "材料包") {
			skipped++
			continue
		}
		filtered = append(filtered, p)
	}
	products = filtered

	if err := os.MkdirAll(h.cfg.UploadDir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create upload directory"})
		return
	}

	var created, updated int
	productIDs := make([]int, 0, len(products))
	for _, p := range products {
		productIDs = append(productIDs, p.ID)
	}
	for _, p := range products {
		var imagePath *string
		if len(p.Images) > 0 && p.Images[0].Src != "" {
			filename, err := downloadImage(r.Context(), p.Images[0].Src, h.cfg.UploadDir, fmt.Sprintf("woo-%d", p.ID))
			if err == nil {
				imagePath = &filename
			}
		}

		// Check existing first so we can accurately report created vs updated.
		var existingID int
		err := h.db.QueryRow(r.Context(),
			"SELECT id FROM courses WHERE woo_product_id = $1",
			p.ID,
		).Scan(&existingID)

		if err != nil {
			// Not found → insert
			_, err := h.db.Exec(r.Context(),
				`INSERT INTO courses (name, image_path, product_url, woo_product_id)
				 VALUES ($1, $2, $3, $4)
				 ON CONFLICT (woo_product_id) DO UPDATE SET
				     name = EXCLUDED.name,
				     product_url = EXCLUDED.product_url,
				     image_path = COALESCE(EXCLUDED.image_path, courses.image_path),
				     updated_at = NOW()`,
				p.Name, imagePath, p.Permalink, p.ID,
			)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to upsert product %d: %v", p.ID, err)})
				return
			}
			created++
		} else {
			// Update
			if imagePath != nil {
				_, err = h.db.Exec(r.Context(),
					`UPDATE courses SET name = $1, product_url = $2, image_path = $3, updated_at = NOW()
					 WHERE woo_product_id = $4`,
					p.Name, p.Permalink, imagePath, p.ID,
				)
			} else {
				_, err = h.db.Exec(r.Context(),
					`UPDATE courses SET name = $1, product_url = $2, updated_at = NOW()
					 WHERE woo_product_id = $3`,
					p.Name, p.Permalink, p.ID,
				)
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to update product %d: %v", p.ID, err)})
				return
			}
			updated++
		}
	}

	var deactivated int64
	if len(productIDs) > 0 {
		tag, err := h.db.Exec(r.Context(),
			`UPDATE courses
			 SET is_active = false, updated_at = NOW()
			 WHERE woo_product_id IS NOT NULL
			   AND woo_product_id != ALL($1)
			   AND is_active = true`,
			productIDs,
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to deactivate stale courses: %v", err)})
			return
		}
		deactivated = tag.RowsAffected()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"synced":      created + updated,
		"created":     created,
		"updated":     updated,
		"deactivated": deactivated,
		"skipped":     skipped,
		"category":    category.Name,
	})
}

// downloadImage fetches the remote image and saves it to uploadDir as
// <basename>.<ext>. Existing files are overwritten.
func downloadImage(ctx context.Context, url, uploadDir, basename string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download image: status %d", resp.StatusCode)
	}

	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "png"):
		ext = ".png"
	case strings.Contains(contentType, "webp"):
		ext = ".webp"
	case strings.Contains(contentType, "gif"):
		ext = ".gif"
	case strings.Contains(contentType, "jpeg"):
		ext = ".jpg"
	default:
		// Fall back to the URL extension if the content type is missing.
		if e := filepath.Ext(url); e != "" && len(e) <= 5 {
			ext = e
		}
	}

	filename := basename + ext
	filePath := filepath.Join(uploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, resp.Body); err != nil {
		return "", fmt.Errorf("save file: %w", err)
	}
	return filename, nil
}
