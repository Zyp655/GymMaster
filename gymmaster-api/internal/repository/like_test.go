package repository

import (
	"context"
	"strings"
	"testing"

	"gymmaster-api/internal/dto"
)

func TestLikeContainsEscapesWildcards(t *testing.T) {
	if got := likeContains(`50%_a\b`); got != `%50\%\_a\\b%` {
		t.Fatalf("got %q", got)
	}
	db := newDryRunDB(t)
	sqls := captureSQL(t, db)
	repo := NewMemberRepository(db)
	_, _, _ = repo.List(context.Background(), dto.Query{Keyword: "%", Page: 1, Limit: 10}, MemberListFilter{})
	joined := strings.Join(*sqls, "\n")
	if !strings.Contains(joined, `%\%%`) {
		t.Fatalf("từ khóa %% phải được thoát: %s", joined)
	}
}
