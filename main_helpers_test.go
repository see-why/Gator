package main

import (
    "context"
    "net/http"
    "testing"
    "strings"
    "database/sql"

    "gator/internal/database"
    "gator/internal/rss"
    "github.com/google/uuid"
)

func TestRenderPosts_TruncationAndPagination(t *testing.T) {
    longDesc := "a"
    for i := 0; i < 300; i++ { longDesc += "a" }

    views := []PostView{
        {Title: "One", Url: "u1", FeedName: "F1", Description: sql.NullString{String: longDesc, Valid: true}, PublishedAt: sql.NullTime{Valid: false}},
        {Title: "Two", Url: "u2", FeedName: "F2"},
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

    client := &http.Client{}
    processed, total := aggregateFeeds(feeds, 2, fetch, save, client, nil)
    if processed != 2 {
        t.Fatalf("expected processed 2, got %d", processed)
    }
    if total != 2 {
        t.Fatalf("expected total 2, got %d", total)
    }
}
