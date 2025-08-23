package main

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"testing"

	"gator/internal/database"
	"gator/internal/rss"

	"github.com/google/uuid"
)

func TestRenderPosts_TruncationAndPagination(t *testing.T) {
	longDesc := "a"
	for i := 0; i < 300; i++ {
		longDesc += "a"
	}

	views := []PostView{
		{ID: uuid.New(), Title: "One", Url: "u1", FeedName: "F1", Description: sql.NullString{String: longDesc, Valid: true}, PublishedAt: sql.NullTime{Valid: false}},
		{ID: uuid.New(), Title: "Two", Url: "u2", FeedName: "F2"},
	}

	out := renderPosts(views, 1, 1) // postsPerPage = 1 -> will show 1 post and hint for more
	if out == "" {
		t.Fatalf("expected non-empty output")
	}
	if !strings.Contains(out, "To see more posts") {
		t.Fatalf("expected pagination hint")
	}
}

func TestAggregateFeeds_WithFakes(t *testing.T) {
	feeds := []database.GetFeedsWithUsersRow{
		{ID: uuid.New(), Name: "a", Url: "u1"},
		{ID: uuid.New(), Name: "b", Url: "u2"},
	}

	fetch := func(ctx context.Context, client *http.Client, url string) (*rss.RSSFeed, error) {
		return &rss.RSSFeed{Channel: rss.RSSChannel{Items: []rss.RSSItem{{Title: "t1", Link: "l1"}}}}, nil
	}
	save := func(ctx context.Context, db *database.Queries, feed *rss.RSSFeed, feedID uuid.UUID) error {
		return nil
	}

	config := AggregationConfig{
		Workers: 2,
		Fetch:   fetch,
		Save:    save,
		Client:  &http.Client{},
		DB:      nil,
	}

	ctx := context.Background()
	result := aggregateFeeds(ctx, feeds, config)

	if result.FeedsProcessed != 2 {
		t.Fatalf("expected FeedsProcessed 2, got %d", result.FeedsProcessed)
	}
	if result.TotalPosts != 2 {
		t.Fatalf("expected TotalPosts 2, got %d", result.TotalPosts)
	}
}

func TestAggregationConfig_DefaultValues(t *testing.T) {
	config := AggregationConfig{
		Workers: 0, // invalid, should be set to 1
		// Fetch and Save are nil, should be set to defaults
		Client: &http.Client{},
		DB:     nil,
	}

	validateConfig(&config)

	if config.Workers != 1 {
		t.Fatalf("expected Workers to be set to 1, got %d", config.Workers)
	}
	if config.Fetch == nil {
		t.Fatalf("expected Fetch to be set to default")
	}
	if config.Save == nil {
		t.Fatalf("expected Save to be set to default")
	}
}
