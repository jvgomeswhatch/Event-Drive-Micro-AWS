package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler { return &AuthHandler{} }

type tokenRequest struct {
	CustomerID string `json:"customerId"`
	Name       string `json:"name"`
}

func (h *AuthHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	var req tokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CustomerID == "" {
		jsonError(w, "customerId is required", http.StatusBadRequest)
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		jsonError(w, "server misconfiguration", http.StatusInternalServerError)
		return
	}

	expiryStr := os.Getenv("JWT_EXPIRY")
	expiry, _ := strconv.Atoi(expiryStr)
	if expiry == 0 {
		expiry = 3600
	}

	claims := jwt.MapClaims{
		"sub":  req.CustomerID,
		"name": req.Name,
		"iss":  "order-service-dev",
		"exp":  time.Now().Add(time.Duration(expiry) * time.Second).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		jsonError(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]any{"token": signed, "expiresIn": expiry}, http.StatusOK)
}
