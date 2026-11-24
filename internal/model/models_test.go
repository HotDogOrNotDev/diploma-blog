package model

import (
	"testing"
	"time"
)

func TestPostIsScheduled(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	tests := []struct {
		name string
		post Post
		want bool
	}{
		{
			name: "draft with future publish_at",
			post: Post{Status: PostStatusDraft, PublishAt: &future},
			want: true,
		},
		{
			name: "draft with past publish_at",
			post: Post{Status: PostStatusDraft, PublishAt: &past},
			want: false,
		},
		{
			name: "published with future publish_at",
			post: Post{Status: PostStatusPublished, PublishAt: &future},
			want: false,
		},
		{
			name: "draft without publish_at",
			post: Post{Status: PostStatusDraft, PublishAt: nil},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.post.IsScheduled(); got != tt.want {
				t.Fatalf("IsScheduled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostShouldPublishNow(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Minute)

	tests := []struct {
		name string
		post Post
		want bool
	}{
		{
			name: "draft ready to publish",
			post: Post{Status: PostStatusDraft, PublishAt: &past},
			want: true,
		},
		{
			name: "draft still waiting",
			post: Post{Status: PostStatusDraft, PublishAt: &future},
			want: false,
		},
		{
			name: "already published",
			post: Post{Status: PostStatusPublished, PublishAt: &past},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.post.ShouldPublishNow(); got != tt.want {
				t.Fatalf("ShouldPublishNow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPostPermissions(t *testing.T) {
	post := Post{AuthorID: 7}
	if !post.CanBeEditedBy(7) || !post.CanBeDeletedBy(7) {
		t.Fatal("author should be able to edit/delete own post")
	}
	if post.CanBeEditedBy(8) || post.CanBeDeletedBy(8) {
		t.Fatal("other user must not edit/delete post")
	}
}

func TestUserCreateRequestValidate(t *testing.T) {
	valid := UserCreateRequest{Username: "alice", Email: "a@b.com", Password: "secret1"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid request, got %v", err)
	}

	invalid := UserCreateRequest{Username: "ab", Email: "bad", Password: "1"}
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
