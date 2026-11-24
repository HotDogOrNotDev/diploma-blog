package apperrors

import (
	"errors"
	"net/http"
)

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPostNotFound       = errors.New("post not found")
	ErrCommentNotFound    = errors.New("comment not found")
	ErrInvalidPostID      = errors.New("invalid post id")
	ErrValidation         = errors.New("validation failed")
	ErrPostNotPublished   = errors.New("post is not published")
)

// HTTPStatus возвращает HTTP-код для известной ошибки приложения.
func HTTPStatus(err error) int {
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrInvalidCredentials):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrPostNotFound), errors.Is(err, ErrCommentNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUserAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidPostID), errors.Is(err, ErrValidation), errors.Is(err, ErrPostNotPublished):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
