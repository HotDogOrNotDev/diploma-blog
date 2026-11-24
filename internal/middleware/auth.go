package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"advanced-blog-management-system/pkg/auth"
)

// UserContextKey используется для сохранения данных пользователя в context
type UserContextKey string

const UserKey UserContextKey = "user"

// AuthMiddleware проверяет JWT токен и добавляет user_id в context.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := ExtractToken(r)
			if err != nil {
				writeUnauthorized(w, "missing or invalid authorization header")
				return
			}

			claims, err := auth.ValidateToken(token, jwtSecret)
			if err != nil {
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware проверяет JWT токен если он присутствует, но не обязателен.
func OptionalAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := ExtractToken(r)
			if err == nil {
				if claims, err := auth.ValidateToken(token, jwtSecret); err == nil {
					ctx := context.WithValue(r.Context(), UserKey, claims.UserID)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserIDFromContext извлекает user_id из context.
func GetUserIDFromContext(r *http.Request) (int, bool) {
	value := r.Context().Value(UserKey)
	if value == nil {
		return 0, false
	}
	userID, ok := value.(int)
	return userID, ok
}

// ExtractToken извлекает JWT токен из заголовка Authorization.
func ExtractToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", errors.New("authorization header is missing")
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("invalid authorization header format")
	}

	return parts[1], nil
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
