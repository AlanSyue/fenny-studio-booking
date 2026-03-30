package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
	"golang.org/x/net/html"
)

type OGImageHandler struct {
	cfg *config.Config
}

func NewOGImageHandler(cfg *config.Config) *OGImageHandler {
	return &OGImageHandler{cfg: cfg}
}

type fetchImageRequest struct {
	URL string `json:"url"`
}

type fetchImageResponse struct {
	ImagePath string `json:"image_path"`
}

func (h *OGImageHandler) FetchFromURL(w http.ResponseWriter, r *http.Request) {
	var req fetchImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Fetch the page
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(req.URL)
	if err != nil {
		http.Error(w, "Failed to fetch URL", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	// Parse HTML to find og:image
	ogImageURL := extractOGImage(resp.Body)
	if ogImageURL == "" {
		http.Error(w, "No og:image found on the page", http.StatusNotFound)
		return
	}

	// Download the image
	imgResp, err := client.Get(ogImageURL)
	if err != nil {
		http.Error(w, "Failed to download image", http.StatusInternalServerError)
		return
	}
	defer imgResp.Body.Close()

	// Determine file extension from Content-Type
	ext := ".jpg"
	contentType := imgResp.Header.Get("Content-Type")
	if strings.Contains(contentType, "png") {
		ext = ".png"
	} else if strings.Contains(contentType, "webp") {
		ext = ".webp"
	} else if strings.Contains(contentType, "gif") {
		ext = ".gif"
	}

	// Save to uploads directory
	if err := os.MkdirAll(h.cfg.UploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(h.cfg.UploadDir, filename)

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, imgResp.Body); err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fetchImageResponse{ImagePath: filename})
}

func extractOGImage(r io.Reader) string {
	tokenizer := html.NewTokenizer(r)
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return ""
		case html.SelfClosingTagToken, html.StartTagToken:
			t := tokenizer.Token()
			if t.Data != "meta" {
				continue
			}
			var property, content string
			for _, attr := range t.Attr {
				if attr.Key == "property" {
					property = attr.Val
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}
			if property == "og:image" && content != "" {
				return content
			}
		}
	}
}
