package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"advanced-blog-management-system/internal/middleware"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/service"

	"github.com/go-chi/chi/v5"
)

type handlerPostRepo struct {
	posts map[int]*model.Post
}

func (r *handlerPostRepo) Create(ctx context.Context, post *model.Post) error { return nil }
func (r *handlerPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	p, ok := r.posts[id]
	if !ok {
		return nil, nil
	}
	cp := *p
	return &cp, nil
}
func (r *handlerPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	return nil, nil
}
func (r *handlerPostRepo) GetTotalCount(ctx context.Context) (int, error) { return 0, nil }
func (r *handlerPostRepo) Update(ctx context.Context, post *model.Post) error {
	cp := *post
	r.posts[post.ID] = &cp
	return nil
}
func (r *handlerPostRepo) Delete(ctx context.Context, id int) error { return nil }
func (r *handlerPostRepo) Exists(ctx context.Context, id int) (bool, error) {
	_, ok := r.posts[id]
	return ok, nil
}
func (r *handlerPostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int, includeDrafts bool) ([]*model.Post, error) {
	return nil, nil
}
func (r *handlerPostRepo) GetTotalCountByAuthorID(ctx context.Context, authorID int, includeDrafts bool) (int, error) {
	return 0, nil
}
func (r *handlerPostRepo) GetScheduledPosts(ctx context.Context) ([]*model.Post, error) {
	return nil, nil
}
func (r *handlerPostRepo) PublishPost(ctx context.Context, id int) error { return nil }

type handlerUserRepo struct{}

func (h *handlerUserRepo) Create(ctx context.Context, user *model.User) error { return nil }
func (h *handlerUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	return &model.User{ID: id, Username: "u", Email: "u@example.com"}, nil
}
func (h *handlerUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, nil
}
func (h *handlerUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return nil, nil
}
func (h *handlerUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (h *handlerUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}
func (h *handlerUserRepo) Update(ctx context.Context, user *model.User) error { return nil }
func (h *handlerUserRepo) Delete(ctx context.Context, id int) error           { return nil }

type handlerCommentRepo struct{}

func (h *handlerCommentRepo) Create(ctx context.Context, comment *model.Comment) error { return nil }
func (h *handlerCommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	return nil, nil
}
func (h *handlerCommentRepo) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	return nil, nil
}
func (h *handlerCommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	return 0, nil
}
func (h *handlerCommentRepo) Update(ctx context.Context, comment *model.Comment) error { return nil }
func (h *handlerCommentRepo) Delete(ctx context.Context, id int) error                 { return nil }

func newTestPostHandler(posts map[int]*model.Post) *PostHandler {
	svc := service.NewPostService(&handlerPostRepo{posts: posts}, &handlerUserRepo{}, &handlerCommentRepo{})
	return NewPostHandler(svc)
}

func withChiURLParam(r *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx))
}

func TestUpdatePostHandlerForbidden(t *testing.T) {
	posts := map[int]*model.Post{
		1: {ID: 1, Title: "t", Content: "c", AuthorID: 1, Status: model.PostStatusPublished},
	}
	h := newTestPostHandler(posts)

	req := httptest.NewRequest(http.MethodPut, "/api/posts/1", bytes.NewBufferString(`{"title":"hack","content":"x"}`))
	req = withChiURLParam(req, "id", "1")
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserKey, 2))

	rr := httptest.NewRecorder()
	h.UpdatePost(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestGetPostHandlerHidesDraft(t *testing.T) {
	future := time.Now().Add(time.Hour)
	posts := map[int]*model.Post{
		5: {ID: 5, Title: "draft", Content: "secret", AuthorID: 1, Status: model.PostStatusDraft, PublishAt: &future},
	}
	h := newTestPostHandler(posts)

	req := httptest.NewRequest(http.MethodGet, "/api/posts/5", nil)
	req = withChiURLParam(req, "id", "5")
	rr := httptest.NewRecorder()
	h.GetPost(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for anonymous draft access, got %d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/posts/5", nil)
	req2 = withChiURLParam(req2, "id", "5")
	req2 = req2.WithContext(context.WithValue(req2.Context(), middleware.UserKey, 1))
	rr2 := httptest.NewRecorder()
	h.GetPost(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("author should get 200, got %d", rr2.Code)
	}
}

func TestParsePaginationDefaultsAndLimits(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	limit, offset := parsePagination(req)
	if limit != 10 || offset != 0 {
		t.Fatalf("defaults: limit=%d offset=%d", limit, offset)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/posts?limit=5&offset=2", nil)
	limit, offset = parsePagination(req)
	if limit != 5 || offset != 2 {
		t.Fatalf("custom: limit=%d offset=%d", limit, offset)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/posts?limit=500&offset=-1", nil)
	limit, offset = parsePagination(req)
	if limit != 100 || offset != 0 {
		t.Fatalf("clamped: limit=%d offset=%d", limit, offset)
	}
}
