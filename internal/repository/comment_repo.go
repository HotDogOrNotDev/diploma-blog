package repository

import (
	"advanced-blog-management-system/internal/model"
	"context"
	"database/sql"
	"fmt"
)

type commentRepository struct {
	db *sql.DB
}

// NewCommentRepository создает новый репозиторий комментариев
func NewCommentRepository(db *sql.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(ctx context.Context, comment *model.Comment) error {
	query := `
		INSERT INTO comments (content, post_id, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, comment.Content, comment.PostID, comment.AuthorID).
		Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (r *commentRepository) GetByID(ctx context.Context, id int) (*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments WHERE id = $1`

	comment := &model.Comment{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&comment.ID, &comment.Content, &comment.PostID, &comment.AuthorID,
		&comment.CreatedAt, &comment.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get comment by id: %w", err)
	}
	return comment, nil
}

func (r *commentRepository) GetByPostID(ctx context.Context, postID int, limit, offset int) ([]*model.Comment, error) {
	query := `
		SELECT id, content, post_id, author_id, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get comments by post: %w", err)
	}
	defer rows.Close()

	comments := make([]*model.Comment, 0)
	for rows.Next() {
		comment := &model.Comment{}
		if err := rows.Scan(
			&comment.ID, &comment.Content, &comment.PostID, &comment.AuthorID,
			&comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comments: %w", err)
	}
	return comments, nil
}

func (r *commentRepository) GetCountByPostID(ctx context.Context, postID int) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM comments WHERE post_id = $1`, postID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get comments count: %w", err)
	}
	return count, nil
}

func (r *commentRepository) Update(ctx context.Context, comment *model.Comment) error {
	query := `
		UPDATE comments
		SET content = $1, updated_at = NOW()
		WHERE id = $2
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query, comment.Content, comment.ID).Scan(&comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	return nil
}

func (r *commentRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM comments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete comment rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
