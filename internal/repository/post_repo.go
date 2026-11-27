package repository

import (
	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
	"context"
	"database/sql"
	"fmt"
)

type postRepository struct {
	db *sql.DB
}

// NewPostRepository создает новый репозиторий постов
func NewPostRepository(db *sql.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *model.Post) error {
	query := `
		INSERT INTO posts (title, content, author_id, status, publish_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		post.Title, post.Content, post.AuthorID, post.Status, post.PublishAt,
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}
	return nil
}

func (r *postRepository) GetByID(ctx context.Context, id int) (*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, status, publish_at, created_at, updated_at
		FROM posts WHERE id = $1`

	post := &model.Post{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&post.ID, &post.Title, &post.Content, &post.AuthorID,
		&post.Status, &post.PublishAt, &post.CreatedAt, &post.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get post by id: %w", err)
	}
	return post, nil
}

func (r *postRepository) GetAll(ctx context.Context, limit, offset int) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, status, publish_at, created_at, updated_at
		FROM posts
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, model.PostStatusPublished, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get all posts: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows)
}

func (r *postRepository) GetTotalCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM posts WHERE status = $1`,
		model.PostStatusPublished,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get posts count: %w", err)
	}
	return count, nil
}

func (r *postRepository) Update(ctx context.Context, post *model.Post) error {
	query := `
		UPDATE posts
		SET title = $1, content = $2, status = $3, publish_at = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at`

	err := r.db.QueryRowContext(
		ctx, query,
		post.Title, post.Content, post.Status, post.PublishAt, post.ID,
	).Scan(&post.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	return nil
}

func (r *postRepository) Delete(ctx context.Context, id int) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete post rows: %w", err)
	}
	if rows == 0 {
		return apperrors.ErrPostNotFound
	}
	return nil
}

func (r *postRepository) Exists(ctx context.Context, id int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("post exists: %w", err)
	}
	return exists, nil
}

func (r *postRepository) GetByAuthorID(ctx context.Context, authorID int, limit, offset int, includeDrafts bool) ([]*model.Post, error) {
	var (
		rows *sql.Rows
		err  error
	)

	if includeDrafts {
		query := `
			SELECT id, title, content, author_id, status, publish_at, created_at, updated_at
			FROM posts
			WHERE author_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3`
		rows, err = r.db.QueryContext(ctx, query, authorID, limit, offset)
	} else {
		query := `
			SELECT id, title, content, author_id, status, publish_at, created_at, updated_at
			FROM posts
			WHERE author_id = $1 AND status = $2
			ORDER BY created_at DESC
			LIMIT $3 OFFSET $4`
		rows, err = r.db.QueryContext(ctx, query, authorID, model.PostStatusPublished, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("get posts by author: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows)
}

func (r *postRepository) GetTotalCountByAuthorID(ctx context.Context, authorID int, includeDrafts bool) (int, error) {
	var count int
	var err error

	if includeDrafts {
		err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM posts WHERE author_id = $1`, authorID).Scan(&count)
	} else {
		err = r.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM posts WHERE author_id = $1 AND status = $2`,
			authorID, model.PostStatusPublished,
		).Scan(&count)
	}
	if err != nil {
		return 0, fmt.Errorf("get posts count by author: %w", err)
	}
	return count, nil
}

func (r *postRepository) GetScheduledPosts(ctx context.Context) ([]*model.Post, error) {
	query := `
		SELECT id, title, content, author_id, status, publish_at, created_at, updated_at
		FROM posts
		WHERE status = $1 AND publish_at IS NOT NULL AND publish_at <= NOW()
		ORDER BY publish_at ASC`

	rows, err := r.db.QueryContext(ctx, query, model.PostStatusDraft)
	if err != nil {
		return nil, fmt.Errorf("get scheduled posts: %w", err)
	}
	defer rows.Close()

	return scanPosts(rows)
}

func (r *postRepository) PublishPost(ctx context.Context, id int) error {
	query := `
		UPDATE posts
		SET status = $1, publish_at = NULL, updated_at = NOW()
		WHERE id = $2 AND status = $3`

	result, err := r.db.ExecContext(ctx, query, model.PostStatusPublished, id, model.PostStatusDraft)
	if err != nil {
		return fmt.Errorf("publish post: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("publish post rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanPosts(rows *sql.Rows) ([]*model.Post, error) {
	posts := make([]*model.Post, 0)
	for rows.Next() {
		post := &model.Post{}
		if err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &post.AuthorID,
			&post.Status, &post.PublishAt, &post.CreatedAt, &post.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, post)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	return posts, nil
}
