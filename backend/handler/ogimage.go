package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
	"golang.org/x/net/html"
)

// Limits for fetching remote pages and images on behalf of an admin.
const (
	maxOGPageBytes   = 2 << 20  // HTML scanned for og:image
	maxOGImageBytes  = 10 << 20 // downloaded image
	ogFetchTimeout   = 15 * time.Second
	maxFetchURLBytes = 2048
)

// allowedImageTypes maps accepted image Content-Types to file extensions.
// SVG is intentionally excluded: it can carry script and is served from our
// own origin under /uploads.
var allowedImageTypes = map[string]string{
	"image/jpeg":  ".jpg",
	"image/jpg":   ".jpg",
	"image/pjpeg": ".jpg",
	"image/png":   ".png",
	"image/webp":  ".webp",
	"image/gif":   ".gif",
}

type OGImageHandler struct {
	cfg    *config.Config
	client *http.Client
	// checkURL validates a URL before it is fetched. Overridable in tests.
	checkURL func(ctx context.Context, u *url.URL) error
}

func NewOGImageHandler(cfg *config.Config) *OGImageHandler {
	return &OGImageHandler{
		cfg:    cfg,
		client: newSafeHTTPClient(ogFetchTimeout),
		checkURL: func(ctx context.Context, u *url.URL) error {
			return checkFetchURL(ctx, u, nil)
		},
	}
}

type fetchImageRequest struct {
	URL string `json:"url"`
}

type fetchImageResponse struct {
	ImagePath string `json:"image_path"`
}

// get validates rawURL (resolved against base when relative) and fetches it
// with the SSRF-safe client.
func (h *OGImageHandler) get(ctx context.Context, base *url.URL, rawURL string) (*http.Response, error) {
	if len(rawURL) > maxFetchURLBytes {
		return nil, errors.New("URL is too long")
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if base != nil {
		u = base.ResolveReference(u)
	}
	if err := h.checkURL(ctx, u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return resp, nil
}

func (h *OGImageHandler) FetchFromURL(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req fetchImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Fetch the page
	resp, err := h.get(ctx, nil, req.URL)
	if err != nil {
		log.Printf("fetch-image: page %q: %v", req.URL, err)
		http.Error(w, "Failed to fetch URL", http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()

	// Parse HTML to find og:image (bounded read)
	ogImageURL := extractOGImage(io.LimitReader(resp.Body, maxOGPageBytes))
	if ogImageURL == "" {
		http.Error(w, "No og:image found on the page", http.StatusNotFound)
		return
	}

	// Download the image; relative og:image URLs resolve against the final
	// page URL (after redirects).
	imgResp, err := h.get(ctx, resp.Request.URL, ogImageURL)
	if err != nil {
		log.Printf("fetch-image: image %q: %v", ogImageURL, err)
		if errors.Is(err, errBlockedDestination) {
			http.Error(w, "Failed to download image", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to download image", http.StatusInternalServerError)
		return
	}
	defer imgResp.Body.Close()

	// Determine file extension from Content-Type; only raster images allowed.
	mediaType, _, err := mime.ParseMediaType(imgResp.Header.Get("Content-Type"))
	ext, ok := allowedImageTypes[strings.ToLower(mediaType)]
	if err != nil || !ok {
		http.Error(w, "og:image is not a supported image type", http.StatusBadRequest)
		return
	}
	if imgResp.ContentLength > maxOGImageBytes {
		http.Error(w, "Image is too large", http.StatusBadRequest)
		return
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

	// Copy at most maxOGImageBytes+1 so an oversized body is detected.
	n, copyErr := io.Copy(dst, io.LimitReader(imgResp.Body, maxOGImageBytes+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || n > maxOGImageBytes {
		os.Remove(filePath)
		if n > maxOGImageBytes {
			http.Error(w, "Image is too large", http.StatusBadRequest)
			return
		}
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
