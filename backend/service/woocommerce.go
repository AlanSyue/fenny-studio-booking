package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
)

type Client struct {
	baseURL   string
	apiKey    string
	apiSecret string
	http      *http.Client
}

func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:   cfg.WooBaseURL,
		apiKey:    cfg.WooAPIKey,
		apiSecret: cfg.WooAPISecret,
		http:      &http.Client{Timeout: 30 * time.Second},
	}
}

type ProductImage struct {
	Src string `json:"src"`
}

type Product struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Permalink string         `json:"permalink"`
	Images    []ProductImage `json:"images"`
}

type Category struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// GetCategoryBySlug fetches a product category by its slug.
// Returns an error if no category matches.
func (c *Client) GetCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return nil, fmt.Errorf("woocommerce credentials not configured")
	}

	url := fmt.Sprintf("%s/wp-json/wc/v3/products/categories?slug=%s", c.baseURL, slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.SetBasicAuth(c.apiKey, c.apiSecret)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call woocommerce: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("woocommerce returned status %d", resp.StatusCode)
	}

	var categories []Category
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, fmt.Errorf("decode categories: %w", err)
	}
	if len(categories) == 0 {
		return nil, fmt.Errorf("no category found for slug %q", slug)
	}
	return &categories[0], nil
}

// ListProductsByCategory fetches published products that belong to the given category ID.
// Pages through all results with per_page=100.
func (c *Client) ListProductsByCategory(ctx context.Context, categoryID int) ([]Product, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return nil, fmt.Errorf("woocommerce credentials not configured")
	}

	const perPage = 100
	var all []Product
	page := 1
	for {
		url := fmt.Sprintf("%s/wp-json/wc/v3/products?category=%d&per_page=%d&status=publish&page=%d",
			c.baseURL, categoryID, perPage, page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.SetBasicAuth(c.apiKey, c.apiSecret)
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("call woocommerce: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("woocommerce returned status %d", resp.StatusCode)
		}

		var batch []Product
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("decode products: %w", err)
		}
		resp.Body.Close()

		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
		page++
	}
	return all, nil
}
