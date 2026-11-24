package handler

import (
	"encoding/json"
	"net/http"

	"advanced-blog-management-system/internal/errors/apperrors"
)

type errorResponse struct {
	Error string `json:"error"`
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, message string, code int) {
	respondWithJSON(w, code, errorResponse{Error: message})
}

func respondWithAppError(w http.ResponseWriter, err error) {
	status := apperrors.HTTPStatus(err)
	respondWithError(w, err.Error(), status)
}
