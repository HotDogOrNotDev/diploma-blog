package service

import (
	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/repository"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type PostService struct {
	postRepo    repository.PostRepository
	userRepo    repository.UserRepository
	commentRepo repository.CommentRepository

	schedulerOnce sync.Once
	schedulerWG   sync.WaitGroup
}

func NewPostService(postRepo repository.PostRepository, userRepo repository.UserRepository, commentRepo repository.CommentRepository) *PostService {
	return &PostService{
		postRepo:    postRepo,
		userRepo:    userRepo,
		commentRepo: commentRepo,
	}
}

func (s *PostService) CreatePost(ctx context.Context, req *model.PostCreateRequest, authorID int) (*model.Post, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	post := &model.Post{
		Title:     req.Title,
		Content:   req.Content,
		AuthorID:  authorID,
		PublishAt: req.PublishAt,
		Status:    model.PostStatusPublished,
	}

	if req.PublishAt != nil && req.PublishAt.After(time.Now()) {
		post.Status = model.PostStatusDraft
	} else {
		post.PublishAt = nil
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) GetPost(ctx context.Context, id int) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperrors.ErrPostNotFound
	}
	return post, nil
}

func (s *PostService) GetAllPosts(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	return s.postRepo.GetAll(ctx, limit, offset)
}

func (s *PostService) GetPostsCount(ctx context.Context) (int, error) {
	return s.postRepo.GetTotalCount(ctx)
}

func (s *PostService) UpdatePost(ctx context.Context, id int, req *model.PostUpdateRequest, userID int) (*model.Post, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperrors.ErrPostNotFound
	}
	if !post.CanBeEditedBy(userID) {
		return nil, apperrors.ErrForbidden
	}

	post.Title = req.Title
	post.Content = req.Content

	if req.PublishAt != nil && req.PublishAt.After(time.Now()) {
		post.Status = model.PostStatusDraft
		post.PublishAt = req.PublishAt
	} else if req.Status != "" {
		post.Status = req.Status
		if req.Status == model.PostStatusPublished {
			post.PublishAt = nil
		} else {
			post.PublishAt = req.PublishAt
		}
	} else if req.PublishAt != nil {
		post.PublishAt = nil
		post.Status = model.PostStatusPublished
	}

	if err := s.postRepo.Update(ctx, post); err != nil {
		return nil, err
	}

	return post, nil
}

func (s *PostService) DeletePost(ctx context.Context, id int, userID int) error {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return apperrors.ErrPostNotFound
	}
	if !post.CanBeDeletedBy(userID) {
		return apperrors.ErrForbidden
	}

	return s.postRepo.Delete(ctx, id)
}

func (s *PostService) GetPostsByAuthor(ctx context.Context, authorID int, limit, offset int) ([]*model.Post, error) {
	return s.postRepo.GetByAuthorID(ctx, authorID, limit, offset)
}

func (s *PostService) GetPostsCountByAuthor(ctx context.Context, authorID int) (int, error) {
	return s.postRepo.GetTotalCountByAuthorID(ctx, authorID)
}

func (s *PostService) BuildPostResponse(ctx context.Context, post *model.Post) (*model.PostResponse, error) {
	author, err := s.userRepo.GetByID(ctx, post.AuthorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, apperrors.ErrUserNotFound
	}

	return &model.PostResponse{
		ID:        post.ID,
		Title:     post.Title,
		Content:   post.Content,
		Author:    author.ToResponse(),
		Status:    post.Status,
		PublishAt: post.PublishAt,
		CreatedAt: post.CreatedAt,
		UpdatedAt: post.UpdatedAt,
	}, nil
}

func (s *PostService) BuildPostResponses(ctx context.Context, posts []*model.Post) ([]model.PostResponse, error) {
	result := make([]model.PostResponse, 0, len(posts))
	for _, post := range posts {
		resp, err := s.BuildPostResponse(ctx, post)
		if err != nil {
			return nil, err
		}
		result = append(result, *resp)
	}
	return result, nil
}

// StartPublisher запускает фоновый планировщик отложенных публикаций.
// interval — период опроса, workers — размер пула воркеров.
// Завершается при отмене ctx (graceful shutdown).
func (s *PostService) StartPublisher(ctx context.Context, interval time.Duration, workers int) {
	s.schedulerOnce.Do(func() {
		if interval <= 0 {
			interval = 10 * time.Second
		}
		if workers <= 0 {
			workers = 4
		}

		jobs := make(chan int, 64)

		for i := 0; i < workers; i++ {
			s.schedulerWG.Add(1)
			go s.publishWorker(ctx, i+1, jobs)
		}

		s.schedulerWG.Add(1)
		go s.schedulerLoop(ctx, interval, jobs)
	})
}

// WaitPublisher ждёт завершения планировщика и воркеров.
func (s *PostService) WaitPublisher() {
	s.schedulerWG.Wait()
}

func (s *PostService) schedulerLoop(ctx context.Context, interval time.Duration, jobs chan<- int) {
	defer s.schedulerWG.Done()
	defer close(jobs)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("publisher scheduler started (interval=%s)", interval)
	s.enqueueDuePosts(ctx, jobs)

	for {
		select {
		case <-ctx.Done():
			log.Println("publisher scheduler stopping...")
			return
		case <-ticker.C:
			s.enqueueDuePosts(ctx, jobs)
		}
	}
}

func (s *PostService) enqueueDuePosts(ctx context.Context, jobs chan<- int) {
	posts, err := s.postRepo.GetScheduledPosts(ctx)
	if err != nil {
		log.Printf("publisher: failed to fetch scheduled posts: %v", err)
		return
	}
	if len(posts) == 0 {
		return
	}

	log.Printf("publisher: found %d post(s) ready to publish", len(posts))
	for _, post := range posts {
		select {
		case <-ctx.Done():
			return
		case jobs <- post.ID:
		}
	}
}

func (s *PostService) publishWorker(ctx context.Context, workerID int, jobs <-chan int) {
	defer s.schedulerWG.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case postID, ok := <-jobs:
			if !ok {
				return
			}
			publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			err := s.postRepo.PublishPost(publishCtx, postID)
			cancel()
			if err != nil {
				log.Printf("publisher worker-%d: failed to publish post %d: %v", workerID, postID, err)
				continue
			}
			log.Printf("publisher worker-%d: published post %d", workerID, postID)
		}
	}
}

// PublishDuePosts публикует все готовые посты синхронно (удобно для тестов).
func (s *PostService) PublishDuePosts(ctx context.Context) (int, error) {
	posts, err := s.postRepo.GetScheduledPosts(ctx)
	if err != nil {
		return 0, err
	}

	published := 0
	for _, post := range posts {
		if err := s.postRepo.PublishPost(ctx, post.ID); err != nil {
			return published, err
		}
		published++
		log.Printf("published scheduled post %d", post.ID)
	}
	return published, nil
}
