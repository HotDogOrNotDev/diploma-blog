package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"advanced-blog-management-system/internal/middleware"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/service"

	"github.com/go-chi/chi/v5"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r)
	if !ok {
		respondWithError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil || postID <= 0 {
		respondWithError(w, "invalid post id", http.StatusBadRequest)
		return
	}

	var req model.CommentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.CreateComment(r.Context(), &req, postID, userID)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	resp, err := h.commentService.BuildCommentResponse(r.Context(), comment)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, resp)
}

func (h *CommentHandler) GetCommentsByPostID(w http.ResponseWriter, r *http.Request) {
	postID, err := strconv.Atoi(chi.URLParam(r, "postId"))
	if err != nil || postID <= 0 {
		respondWithError(w, "invalid post id", http.StatusBadRequest)
		return
	}

	limit, offset := parsePagination(r)

	comments, err := h.commentService.GetCommentsByPostID(r.Context(), postID, limit, offset)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	total, err := h.commentService.GetCommentsCountByPostID(r.Context(), postID)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	items, err := h.commentService.BuildCommentResponses(r.Context(), comments)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"comments": items,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *CommentHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r)
	if !ok {
		respondWithError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	commentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || commentID <= 0 {
		respondWithError(w, "invalid comment id", http.StatusBadRequest)
		return
	}

	var req model.CommentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	comment, err := h.commentService.UpdateComment(r.Context(), commentID, &req, userID)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	resp, err := h.commentService.BuildCommentResponse(r.Context(), comment)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r)
	if !ok {
		respondWithError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	commentID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || commentID <= 0 {
		respondWithError(w, "invalid comment id", http.StatusBadRequest)
		return
	}

	if err := h.commentService.DeleteComment(r.Context(), commentID, userID); err != nil {
		respondWithAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
