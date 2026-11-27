package repository

import (
	"context"
	"testing"
	"time"

	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostRepositoryGetAllFiltersPublished(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	repo := NewPostRepository(db)
	rows := sqlmock.NewRows([]string{
		"id", "title", "content", "author_id", "status", "publish_at", "created_at", "updated_at",
	}).AddRow(1, "Pub", "body", 1, model.PostStatusPublished, nil, now, now)

	mock.ExpectQuery(`WHERE status = \$1`).
		WithArgs(model.PostStatusPublished, 10, 0).
		WillReturnRows(rows)

	posts, err := repo.GetAll(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(posts) != 1 || posts[0].Status != model.PostStatusPublished {
		t.Fatalf("unexpected posts: %+v", posts)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostRepositoryGetTotalCountPublishedOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPostRepository(db)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM posts WHERE status = \$1`).
		WithArgs(model.PostStatusPublished).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	count, err := repo.GetTotalCount(context.Background())
	if err != nil {
		t.Fatalf("GetTotalCount: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1, got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostRepositoryDeleteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPostRepository(db)
	mock.ExpectExec(`DELETE FROM posts WHERE id = \$1`).
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = repo.Delete(context.Background(), 42)
	if err != apperrors.ErrPostNotFound {
		t.Fatalf("expected ErrPostNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestPostRepositoryGetByAuthorIDPublishedOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	repo := NewPostRepository(db)
	rows := sqlmock.NewRows([]string{
		"id", "title", "content", "author_id", "status", "publish_at", "created_at", "updated_at",
	}).AddRow(1, "Pub", "body", 7, model.PostStatusPublished, nil, now, now)

	mock.ExpectQuery(`author_id = \$1 AND status = \$2`).
		WithArgs(7, model.PostStatusPublished, 10, 0).
		WillReturnRows(rows)

	posts, err := repo.GetByAuthorID(context.Background(), 7, 10, 0, false)
	if err != nil {
		t.Fatalf("GetByAuthorID: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
