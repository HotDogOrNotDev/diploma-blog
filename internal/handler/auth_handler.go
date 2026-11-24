package handler

import (
	"encoding/json"
	"net/http"

	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/service"
	"advanced-blog-management-system/pkg/auth"
)

type AuthHandler struct {
	userService *service.UserService
	jwtSecret   string
}

func NewAuthHandler(userService *service.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

// RegisterHandler обрабатывает POST /api/register
func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req model.UserCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.Register(r.Context(), &req)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	token, expiresAt, err := auth.GenerateToken(user.ID, user.Email, user.Username, h.jwtSecret)
	if err != nil {
		respondWithError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusCreated, model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	})
}

// LoginHandler обрабатывает POST /api/login
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req model.UserLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	user, err := h.userService.Login(r.Context(), &req)
	if err != nil {
		if err == apperrors.ErrInvalidCredentials {
			respondWithError(w, err.Error(), http.StatusUnauthorized)
			return
		}
		respondWithAppError(w, err)
		return
	}

	token, expiresAt, err := auth.GenerateToken(user.ID, user.Email, user.Username, h.jwtSecret)
	if err != nil {
		respondWithError(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusOK, model.TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	})
}
