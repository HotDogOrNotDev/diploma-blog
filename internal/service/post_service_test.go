package service

import (
	"context"
	"sync"
	"testing"
	"time"

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
	return nil, nil
}

func (m *mockPostRepo) GetTotalCount(ctx context.Context) (int, error) {
	return 0, nil
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

func (m *mockPostRepo) GetByAuthorID(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error) {
	return nil, nil
}

func (m *mockPostRepo) GetTotalCountByAuthorID(ctx context.Context, authorID int) (int, error) {
	return 0, nil
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
