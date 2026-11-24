package service

import (
	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/repository"
	"context"
	"fmt"
)

type CommentService struct {
	commentRepo repository.CommentRepository
	postRepo    repository.PostRepository
	userRepo    repository.UserRepository
}

func NewCommentService(commentRepo repository.CommentRepository, postRepo repository.PostRepository, userRepo repository.UserRepository) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
		userRepo:    userRepo,
	}
}

func (s *CommentService) CreateComment(ctx context.Context, req *model.CommentCreateRequest, postID int, authorID int) (*model.Comment, error) {
	req.PostID = postID
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, apperrors.ErrPostNotFound
	}
	if post.Status != model.PostStatusPublished {
		return nil, apperrors.ErrPostNotPublished
	}

	comment := &model.Comment{
		Content:  req.Content,
		PostID:   postID,
		AuthorID: authorID,
	}

	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) GetComment(ctx context.Context, id int) (*model.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, apperrors.ErrCommentNotFound
	}
	return comment, nil
}

func (s *CommentService) GetCommentsByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	exists, err := s.postRepo.Exists(ctx, postID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperrors.ErrPostNotFound
	}

	return s.commentRepo.GetByPostID(ctx, postID, limit, offset)
}

func (s *CommentService) GetCommentsCountByPostID(ctx context.Context, postID int) (int, error) {
	return s.commentRepo.GetCountByPostID(ctx, postID)
}

func (s *CommentService) UpdateComment(ctx context.Context, id int, req *model.CommentUpdateRequest, userID int) (*model.Comment, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if comment == nil {
		return nil, apperrors.ErrCommentNotFound
	}
	if !comment.CanBeEditedBy(userID) {
		return nil, apperrors.ErrForbidden
	}

	comment.Content = req.Content
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}

	return comment, nil
}

func (s *CommentService) DeleteComment(ctx context.Context, id int, userID int) error {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if comment == nil {
		return apperrors.ErrCommentNotFound
	}
	if !comment.CanBeDeletedBy(userID) {
		return apperrors.ErrForbidden
	}

	return s.commentRepo.Delete(ctx, id)
}

func (s *CommentService) BuildCommentResponse(ctx context.Context, comment *model.Comment) (*model.CommentResponse, error) {
	author, err := s.userRepo.GetByID(ctx, comment.AuthorID)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, apperrors.ErrUserNotFound
	}

	return &model.CommentResponse{
		ID:        comment.ID,
		Content:   comment.Content,
		PostID:    comment.PostID,
		Author:    author.ToResponse(),
		CreatedAt: comment.CreatedAt,
		UpdatedAt: comment.UpdatedAt,
	}, nil
}

func (s *CommentService) BuildCommentResponses(ctx context.Context, comments []*model.Comment) ([]model.CommentResponse, error) {
	result := make([]model.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		resp, err := s.BuildCommentResponse(ctx, comment)
		if err != nil {
			return nil, err
		}
		result = append(result, *resp)
	}
	return result, nil
}
