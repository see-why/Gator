package rss

import (
	"context"
	"encoding/xml"
	"html"
	"io"
	"net/http"
)

// FetchFeed fetches a feed from the given URL and returns a filled-out RSSFeed struct
func FetchFeed(ctx context.Context, client *http.Client, feedURL string) (*RSSFeed, error) {
	// Create a new request with context
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	// Set User-Agent header to identify our program
	req.Header.Set("User-Agent", "gator")

	// Make the request using the provided client
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the XML into RSSFeed struct
	var feed RSSFeed
	err = xml.Unmarshal(body, &feed)
	if err != nil {
		return nil, err
	}

	// Decode HTML entities in channel title and description
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)

	// Decode HTML entities in item titles and descriptions
	for i := range feed.Channel.Items {
		feed.Channel.Items[i].Title = html.UnescapeString(feed.Channel.Items[i].Title)
		feed.Channel.Items[i].Description = html.UnescapeString(feed.Channel.Items[i].Description)
	}

	return &feed, nil
}
