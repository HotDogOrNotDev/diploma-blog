package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
)

type mockPostRepo struct {
	mu        sync.Mutex
	posts     map[int]*model.Post
	nextID    int
	published []int
}

func newMockPostRepo() *mockPostRepo {
	return &mockPostRepo{
		posts:  make(map[int]*model.Post),
		nextID: 1,
	}
}

func (m *mockPostRepo) Create(ctx context.Context, post *model.Post) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	post.ID = m.nextID
	m.nextID++
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now
	cp := *post
	m.posts[post.ID] = &cp
	return nil
}

func (m *mockPostRepo) GetByID(ctx context.Context, id int) (*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	post, ok := m.posts[id]
	if !ok {
		return nil, nil
	}
	cp := *post
	return &cp, nil
}

func (m *mockPostRepo) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*model.Post
	for _, post := range m.posts {
		if post.Status == model.PostStatusPublished {
			cp := *post
			result = append(result, &cp)
		}
	}
	if offset > len(result) {
		return []*model.Post{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, post := range m.posts {
		if post.Status == model.PostStatusPublished {
			count++
		}
	}
	return count, nil
}

func (m *mockPostRepo) Update(ctx context.Context, post *model.Post) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *post
	m.posts[post.ID] = &cp
	return nil
}

func (m *mockPostRepo) Delete(ctx context.Context, id int) error { return nil }

func (m *mockPostRepo) Exists(ctx context.Context, id int) (bool, error) { return false, nil }

func (m *mockPostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int, includeDrafts bool) ([]*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*model.Post
	for _, post := range m.posts {
		if post.AuthorID != authorID {
			continue
		}
		if !includeDrafts && post.Status != model.PostStatusPublished {
			continue
		}
		cp := *post
		result = append(result, &cp)
	}
	return result, nil
}

func (m *mockPostRepo) GetTotalCountByAuthorID(ctx context.Context, authorID int, includeDrafts bool) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, post := range m.posts {
		if post.AuthorID != authorID {
			continue
		}
		if !includeDrafts && post.Status != model.PostStatusPublished {
			continue
		}
		count++
	}
	return count, nil
}

func (m *mockPostRepo) GetScheduledPosts(ctx context.Context) ([]*model.Post, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*model.Post
	now := time.Now()
	for _, post := range m.posts {
		if post.Status == model.PostStatusDraft && post.PublishAt != nil && !post.PublishAt.After(now) {
			cp := *post
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockPostRepo) PublishPost(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	post, ok := m.posts[id]
	if !ok {
		return nil
	}
	post.Status = model.PostStatusPublished
	post.PublishAt = nil
	m.published = append(m.published, id)
	return nil
}

type stubUserRepo struct{}

func (s *stubUserRepo) Create(ctx context.Context, user *model.User) error { return nil }
func (s *stubUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	return &model.User{ID: id, Username: "u", Email: "u@example.com"}, nil
}
func (s *stubUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return nil, nil
}
func (s *stubUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return nil, nil
}
func (s *stubUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (s *stubUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return false, nil
}
func (s *stubUserRepo) Update(ctx context.Context, user *model.User) error { return nil }
func (s *stubUserRepo) Delete(ctx context.Context, id int) error           { return nil }

type stubCommentRepo struct{}

func (s *stubCommentRepo) Create(ctx context.Context, comment *model.Comment) error { return nil }
func (s *stubCommentRepo) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	return nil, nil
}
func (s *stubCommentRepo) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	return nil, nil
}
func (s *stubCommentRepo) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	return 0, nil
}
func (s *stubCommentRepo) Update(ctx context.Context, comment *model.Comment) error { return nil }
func (s *stubCommentRepo) Delete(ctx context.Context, id int) error                 { return nil }

func TestCreatePostStatusByPublishAt(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	immediate, err := svc.CreatePost(ctx, &model.PostCreateRequest{
		Title:   "now",
		Content: "body",
	}, 1)
	if err != nil {
		t.Fatalf("CreatePost immediate: %v", err)
	}
	if immediate.Status != model.PostStatusPublished {
		t.Fatalf("expected published, got %s", immediate.Status)
	}

	future := time.Now().Add(2 * time.Hour)
	scheduled, err := svc.CreatePost(ctx, &model.PostCreateRequest{
		Title:     "later",
		Content:   "body",
		PublishAt: &future,
	}, 1)
	if err != nil {
		t.Fatalf("CreatePost scheduled: %v", err)
	}
	if scheduled.Status != model.PostStatusDraft {
		t.Fatalf("expected draft, got %s", scheduled.Status)
	}
	if scheduled.PublishAt == nil {
		t.Fatal("expected publish_at to be set")
	}
}

func TestPublishDuePostsAfterPublishAt(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	past := time.Now().Add(-time.Minute)
	post, err := svc.CreatePost(ctx, &model.PostCreateRequest{
		Title:     "due",
		Content:   "body",
		PublishAt: &past,
	}, 1)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	// CreatePost treats past publish_at as immediate publish; force draft for scheduler test.
	repo.mu.Lock()
	repo.posts[post.ID].Status = model.PostStatusDraft
	repo.posts[post.ID].PublishAt = &past
	repo.mu.Unlock()

	n, err := svc.PublishDuePosts(ctx)
	if err != nil {
		t.Fatalf("PublishDuePosts: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 published post, got %d", n)
	}

	updated, err := repo.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Status != model.PostStatusPublished {
		t.Fatalf("expected published status, got %s", updated.Status)
	}
	if updated.PublishAt != nil {
		t.Fatal("publish_at must be cleared after publish")
	}
}

func TestGetPostHidesDraftFromOthers(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	future := time.Now().Add(time.Hour)
	draft, err := svc.CreatePost(ctx, &model.PostCreateRequest{
		Title: "secret", Content: "draft", PublishAt: &future,
	}, 1)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	if _, err := svc.GetPost(ctx, draft.ID, 0); !errors.Is(err, apperrors.ErrPostNotFound) {
		t.Fatalf("anonymous should get not found, got %v", err)
	}
	if _, err := svc.GetPost(ctx, draft.ID, 2); !errors.Is(err, apperrors.ErrPostNotFound) {
		t.Fatalf("other user should get not found, got %v", err)
	}
	got, err := svc.GetPost(ctx, draft.ID, 1)
	if err != nil || got.Status != model.PostStatusDraft {
		t.Fatalf("author should see draft, err=%v post=%v", err, got)
	}
}

func TestGetAllPostsExcludesDrafts(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	_, _ = svc.CreatePost(ctx, &model.PostCreateRequest{Title: "pub", Content: "x"}, 1)
	future := time.Now().Add(time.Hour)
	_, _ = svc.CreatePost(ctx, &model.PostCreateRequest{Title: "draft", Content: "y", PublishAt: &future}, 1)

	posts, err := svc.GetAllPosts(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetAllPosts: %v", err)
	}
	if len(posts) != 1 || posts[0].Status != model.PostStatusPublished {
		t.Fatalf("expected only published posts, got %+v", posts)
	}

	total, err := svc.GetPostsCount(ctx)
	if err != nil {
		t.Fatalf("GetPostsCount: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total=1, got %d", total)
	}
}

func TestGetPostsByAuthorDraftVisibility(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	_, _ = svc.CreatePost(ctx, &model.PostCreateRequest{Title: "pub", Content: "x"}, 1)
	future := time.Now().Add(time.Hour)
	_, _ = svc.CreatePost(ctx, &model.PostCreateRequest{Title: "draft", Content: "y", PublishAt: &future}, 1)

	publicView, err := svc.GetPostsByAuthor(ctx, 1, 0, 10, 0)
	if err != nil {
		t.Fatalf("public view: %v", err)
	}
	if len(publicView) != 1 {
		t.Fatalf("stranger should see 1 published, got %d", len(publicView))
	}

	ownerView, err := svc.GetPostsByAuthor(ctx, 1, 1, 10, 0)
	if err != nil {
		t.Fatalf("owner view: %v", err)
	}
	if len(ownerView) != 2 {
		t.Fatalf("owner should see drafts too, got %d", len(ownerView))
	}
}

func TestGetAllPostsPagination(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := svc.CreatePost(ctx, &model.PostCreateRequest{Title: "p", Content: "c"}, 1)
		if err != nil {
			t.Fatalf("CreatePost: %v", err)
		}
	}

	page, err := svc.GetAllPosts(ctx, 2, 0)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected limit=2, got %d", len(page))
	}

	page2, err := svc.GetAllPosts(ctx, 2, 2)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("expected remaining 1, got %d", len(page2))
	}
}

func TestUpdatePostForbiddenForNonAuthor(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	post, err := svc.CreatePost(ctx, &model.PostCreateRequest{Title: "t", Content: "c"}, 1)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	title := "hack"
	_, err = svc.UpdatePost(ctx, post.ID, &model.PostUpdateRequest{Title: &title}, 2)
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("expected forbidden, got %v", err)
	}
}

func TestUpdatePostPartial(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})
	ctx := context.Background()

	post, err := svc.CreatePost(ctx, &model.PostCreateRequest{Title: "old", Content: "body"}, 1)
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}

	title := "new"
	updated, err := svc.UpdatePost(ctx, post.ID, &model.PostUpdateRequest{Title: &title}, 1)
	if err != nil {
		t.Fatalf("UpdatePost: %v", err)
	}
	if updated.Title != "new" || updated.Content != "body" {
		t.Fatalf("partial update failed: %+v", updated)
	}
}

func TestPublisherWorkerPool(t *testing.T) {
	repo := newMockPostRepo()
	svc := NewPostService(repo, &stubUserRepo{}, &stubCommentRepo{})

	past := time.Now().Add(-time.Second)
	repo.mu.Lock()
	repo.posts[1] = &model.Post{ID: 1, Status: model.PostStatusDraft, PublishAt: &past, Title: "a", Content: "b", AuthorID: 1}
	repo.posts[2] = &model.Post{ID: 2, Status: model.PostStatusDraft, PublishAt: &past, Title: "c", Content: "d", AuthorID: 1}
	repo.nextID = 3
	repo.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	svc.StartPublisher(ctx, 50*time.Millisecond, 2)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		done := len(repo.published) >= 2
		repo.mu.Unlock()
		if done {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	svc.WaitPublisher()

	repo.mu.Lock()
	defer repo.mu.Unlock()
	if len(repo.published) < 2 {
		t.Fatalf("expected both posts published, got %v", repo.published)
	}
	if repo.posts[1].Status != model.PostStatusPublished || repo.posts[2].Status != model.PostStatusPublished {
		t.Fatal("both posts must become published")
	}
}
