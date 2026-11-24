package handler

import "net/http"

// HealthCheckHandler обрабатывает GET /api/health
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
