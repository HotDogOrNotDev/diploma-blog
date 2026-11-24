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

type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r)
	if !ok {
		respondWithError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.PostCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	post, err := h.postService.CreatePost(r.Context(), &req, userID)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	resp, err := h.postService.BuildPostResponse(r.Context(), post)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusCreated, resp)
}

func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		respondWithError(w, "invalid post id", http.StatusBadRequest)
		return
	}

	post, err := h.postService.GetPost(r.Context(), id)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	resp, err := h.postService.BuildPostResponse(r.Context(), post)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (h *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r)

	posts, err := h.postService.GetAllPosts(r.Context(), limit, offset)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	total, err := h.postService.GetPostsCount(r.Context())
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	items, err := h.postService.BuildPostResponses(r.Context(), posts)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"posts":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r)
	if !ok {
		respondWithError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		respondWithError(w, "invalid post id", http.StatusBadRequest)
		return
	}

	var req model.PostUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	post, err := h.postService.UpdatePost(r.Context(), id, &req, userID)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	resp, err := h.postService.BuildPostResponse(r.Context(), post)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r)
	if !ok {
		respondWithError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		respondWithError(w, "invalid post id", http.StatusBadRequest)
		return
	}

	if err := h.postService.DeletePost(r.Context(), id, userID); err != nil {
		respondWithAppError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PostHandler) GetPostsByAuthor(w http.ResponseWriter, r *http.Request) {
	authorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || authorID <= 0 {
		respondWithError(w, "invalid user id", http.StatusBadRequest)
		return
	}

	limit, offset := parsePagination(r)

	posts, err := h.postService.GetPostsByAuthor(r.Context(), authorID, limit, offset)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	total, err := h.postService.GetPostsCountByAuthor(r.Context(), authorID)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	items, err := h.postService.BuildPostResponses(r.Context(), posts)
	if err != nil {
		respondWithAppError(w, err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"posts":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func parsePagination(r *http.Request) (limit, offset int) {
	limit = 10
	offset = 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	return limit, offset
}
