package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
	"github.com/AlanSyue/fenny-studio-booking/backend/db"
	"github.com/AlanSyue/fenny-studio-booking/backend/handler"
	"github.com/AlanSyue/fenny-studio-booking/backend/middleware"
	"github.com/AlanSyue/fenny-studio-booking/backend/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env if present (silently ignore if not found — production will use real env vars)
	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	wooClient := service.New(cfg)
	telegramNotifier := service.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID)

	authHandler := handler.NewAuthHandler(pool, cfg)
	courseHandler := handler.NewCourseHandler(pool, cfg, wooClient)
	inquiryHandler := handler.NewInquiryHandler(pool, telegramNotifier)
	ogImageHandler := handler.NewOGImageHandler(cfg)

	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	// In-app per-IP rate limits (defense in depth behind nginx limit_req).
	loginLimiter := middleware.NewIPRateLimiter(5, 5)   // 5/min, burst 5
	inquiryLimiter := middleware.NewIPRateLimiter(3, 5) // 3/min, burst 5
	r.With(loginLimiter.Middleware).Post("/api/auth/login", authHandler.Login)
	r.Get("/api/courses", courseHandler.ListActive)
	r.With(inquiryLimiter.Middleware).Post("/api/inquiries", inquiryHandler.Create)

	// Admin routes (JWT protected)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(middleware.JWTAuth(cfg.JWTSecret))

		r.Get("/courses", courseHandler.ListAll)
		r.Post("/courses", courseHandler.Create)
		r.Put("/courses/{id}", courseHandler.Update)
		r.Delete("/courses/{id}", courseHandler.Delete)
		r.Post("/courses/sync-woocommerce", courseHandler.SyncFromWoo)

		r.Post("/fetch-image", ogImageHandler.FetchFromURL)

		r.Get("/inquiries", inquiryHandler.List)
		r.Patch("/inquiries/{id}", inquiryHandler.ToggleProcessed)
	})

	// Serve uploaded files (no directory listings)
	fileServer := http.FileServer(handler.NoDirFileSystem{FS: http.Dir(cfg.UploadDir)})
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", fileServer))

	log.Printf("Server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", cfg.Port), r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
