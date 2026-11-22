package handler

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/AlanSyue/fenny-studio-booking/backend/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db  *pgxpool.Pool
	cfg *config.Config
}

func NewAuthHandler(db *pgxpool.Pool, cfg *config.Config) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

// maxLoginBodyBytes bounds the login request body; credentials are tiny.
const maxLoginBodyBytes = 16 << 10

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// dummyPasswordHash returns a bcrypt hash (DefaultCost, like real admin
// hashes) that no submitted password can match. Comparing against it when the
// username does not exist makes unknown-user failures take about as long as
// wrong-password failures, so usernames cannot be enumerated by timing.
func dummyPasswordHash() []byte {
	dummyHashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte("fenny-login-timing-equalizer"), bcrypt.DefaultCost)
		if err != nil {
			panic(err) // only fails for over-long passwords or bad cost
		}
		dummyHash = h
	})
	return dummyHash
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBodyBytes)
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var id int
	var passwordHash string
	err := h.db.QueryRow(r.Context(),
		"SELECT id, password_hash FROM admin_users WHERE username = $1", req.Username,
	).Scan(&id, &passwordHash)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(req.Password))
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  id,
		"username": req.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{Token: tokenString})
}
