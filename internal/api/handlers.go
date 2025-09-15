package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"gator/internal/database"
	"gator/internal/rss"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// User handlers
func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get users")
		return
	}

	type userResponse struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	response := make([]userResponse, len(users))
	for i, user := range users {
		response[i] = userResponse{
			ID:        user.ID,
			Name:      user.Name,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	s.respondWithJSON(w, http.StatusOK, user)
}

// Feed handlers
func (s *Server) handleGetFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.db.GetFeedsWithUsers(context.Background())
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get feeds")
		return
	}

	type feedResponse struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		URL       string    `json:"url"`
		UserName  string    `json:"user_name"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	response := make([]feedResponse, len(feeds))
	for i, feed := range feeds {
		response[i] = feedResponse{
			ID:        feed.ID,
			Name:      feed.Name,
			URL:       feed.Url,
			UserName:  feed.UserName,
			CreatedAt: feed.CreatedAt,
			UpdatedAt: feed.UpdatedAt,
		}
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

type createFeedRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req createFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Name == "" || req.URL == "" {
		s.respondWithError(w, http.StatusBadRequest, "Name and URL are required")
		return
	}

	// Create feed
	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      req.Name,
		Url:       req.URL,
		UserID:    user.ID,
	})
	if err != nil {
		s.respondWithError(w, http.StatusConflict, "Feed already exists")
		return
	}

	// Auto-follow the feed
	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to auto-follow feed")
		return
	}

	// Fetch and save posts in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := rss.NewHTTPClient()
		rssFeed, err := rss.FetchFeed(ctx, client, req.URL)
		if err == nil {
			rss.SavePostsToDatabase(ctx, s.db, rssFeed, feed.ID)
		}
	}()

	type feedResponse struct {
		ID        uuid.UUID `json:"id"`
		Name      string    `json:"name"`
		URL       string    `json:"url"`
		UserID    uuid.UUID `json:"user_id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	response := feedResponse{
		ID:        feed.ID,
		Name:      feed.Name,
		URL:       feed.Url,
		UserID:    feed.UserID,
		CreatedAt: feed.CreatedAt,
		UpdatedAt: feed.UpdatedAt,
	}

	s.respondWithJSON(w, http.StatusCreated, response)
}

// Feed follow handlers
func (s *Server) handleGetFeedFollows(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get feed follows")
		return
	}

	type feedFollowResponse struct {
		ID        uuid.UUID `json:"id"`
		FeedName  string    `json:"feed_name"`
		CreatedAt time.Time `json:"created_at"`
	}

	response := make([]feedFollowResponse, len(follows))
	for i, follow := range follows {
		response[i] = feedFollowResponse{
			ID:        follow.ID,
			FeedName:  follow.FeedName,
			CreatedAt: follow.CreatedAt,
		}
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

type createFeedFollowRequest struct {
	FeedURL string `json:"feed_url"`
}

func (s *Server) handleCreateFeedFollow(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req createFeedFollowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.FeedURL == "" {
		s.respondWithError(w, http.StatusBadRequest, "Feed URL is required")
		return
	}

	// Get feed by URL
	feed, err := s.db.GetFeedByURL(context.Background(), req.FeedURL)
	if err != nil {
		s.respondWithError(w, http.StatusNotFound, "Feed not found")
		return
	}

	// Create follow
	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		s.respondWithError(w, http.StatusConflict, "Already following this feed")
		return
	}

	type feedFollowResponse struct {
		ID        uuid.UUID `json:"id"`
		FeedName  string    `json:"feed_name"`
		CreatedAt time.Time `json:"created_at"`
	}

	response := feedFollowResponse{
		ID:        follow.ID,
		FeedName:  follow.FeedName,
		CreatedAt: follow.CreatedAt,
	}

	s.respondWithJSON(w, http.StatusCreated, response)
}

type deleteFeedFollowRequest struct {
	FeedURL string `json:"feed_url"`
}

func (s *Server) handleDeleteFeedFollow(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req deleteFeedFollowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.FeedURL == "" {
		s.respondWithError(w, http.StatusBadRequest, "Feed URL is required")
		return
	}

	rowsAffected, err := s.db.DeleteFeedFollowByUserAndFeedURL(context.Background(), database.DeleteFeedFollowByUserAndFeedURLParams{
		UserID: user.ID,
		Url:    req.FeedURL,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to unfollow feed")
		return
	}

	if rowsAffected == 0 {
		s.respondWithError(w, http.StatusNotFound, "Not following this feed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Post handlers
func (s *Server) handleGetPosts(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page := int32(1)
	limit := int32(10)

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = int32(p)
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	offset := (page - 1) * limit

	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get posts")
		return
	}

	type postResponse struct {
		ID          uuid.UUID  `json:"id"`
		Title       string     `json:"title"`
		URL         string     `json:"url"`
		Description *string    `json:"description"`
		PublishedAt *time.Time `json:"published_at"`
		FeedName    string     `json:"feed_name"`
		CreatedAt   time.Time  `json:"created_at"`
	}

	response := make([]postResponse, len(posts))
	for i, post := range posts {
		var description *string
		if post.Description.Valid {
			description = &post.Description.String
		}

		var publishedAt *time.Time
		if post.PublishedAt.Valid {
			publishedAt = &post.PublishedAt.Time
		}

		response[i] = postResponse{
			ID:          post.ID,
			Title:       post.Title,
			URL:         post.Url,
			Description: description,
			PublishedAt: publishedAt,
			FeedName:    post.FeedName,
			CreatedAt:   post.CreatedAt,
		}
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

func (s *Server) handleSearchPosts(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		s.respondWithError(w, http.StatusBadRequest, "Search query 'q' is required")
		return
	}

	// Parse pagination parameters
	page := int32(1)
	limit := int32(10)

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = int32(p)
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	offset := (page - 1) * limit

	posts, err := s.db.SearchPostsForUser(context.Background(), database.SearchPostsForUserParams{
		UserID:  user.ID,
		Column2: sql.NullString{String: query, Valid: true},
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to search posts")
		return
	}

	type postResponse struct {
		ID          uuid.UUID  `json:"id"`
		Title       string     `json:"title"`
		URL         string     `json:"url"`
		Description *string    `json:"description"`
		PublishedAt *time.Time `json:"published_at"`
		FeedName    string     `json:"feed_name"`
		CreatedAt   time.Time  `json:"created_at"`
	}

	response := make([]postResponse, len(posts))
	for i, post := range posts {
		var description *string
		if post.Description.Valid {
			description = &post.Description.String
		}

		var publishedAt *time.Time
		if post.PublishedAt.Valid {
			publishedAt = &post.PublishedAt.Time
		}

		response[i] = postResponse{
			ID:          post.ID,
			Title:       post.Title,
			URL:         post.Url,
			Description: description,
			PublishedAt: publishedAt,
			FeedName:    post.FeedName,
			CreatedAt:   post.CreatedAt,
		}
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

// Bookmark handlers
func (s *Server) handleGetBookmarks(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Parse pagination parameters
	page := int32(1)
	limit := int32(10)

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = int32(p)
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = int32(l)
		}
	}

	offset := (page - 1) * limit

	bookmarks, err := s.db.GetBookmarksForUser(context.Background(), database.GetBookmarksForUserParams{
		UserID: user.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to get bookmarks")
		return
	}

	type bookmarkResponse struct {
		PostID       uuid.UUID  `json:"post_id"`
		Title        string     `json:"title"`
		URL          string     `json:"url"`
		Description  *string    `json:"description"`
		PublishedAt  *time.Time `json:"published_at"`
		FeedName     string     `json:"feed_name"`
		BookmarkedAt time.Time  `json:"bookmarked_at"`
	}

	response := make([]bookmarkResponse, len(bookmarks))
	for i, bookmark := range bookmarks {
		var description *string
		if bookmark.Description.Valid {
			description = &bookmark.Description.String
		}

		var publishedAt *time.Time
		if bookmark.PublishedAt.Valid {
			publishedAt = &bookmark.PublishedAt.Time
		}

		response[i] = bookmarkResponse{
			PostID:       bookmark.ID,
			Title:        bookmark.Title,
			URL:          bookmark.Url,
			Description:  description,
			PublishedAt:  publishedAt,
			FeedName:     bookmark.FeedName,
			BookmarkedAt: bookmark.BookmarkedAt,
		}
	}

	s.respondWithJSON(w, http.StatusOK, response)
}

type createBookmarkRequest struct {
	PostID string `json:"post_id"`
}

func (s *Server) handleCreateBookmark(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req createBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid post ID format")
		return
	}

	// Check if post exists
	_, err = s.db.GetPostByID(context.Background(), postID)
	if err != nil {
		s.respondWithError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Create bookmark
	bookmark, err := s.db.CreateBookmark(context.Background(), database.CreateBookmarkParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    user.ID,
		PostID:    postID,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to create bookmark")
		return
	}

	if bookmark.ID == uuid.Nil {
		s.respondWithError(w, http.StatusConflict, "Post already bookmarked")
		return
	}

	type bookmarkResponse struct {
		ID        uuid.UUID `json:"id"`
		PostID    uuid.UUID `json:"post_id"`
		UserID    uuid.UUID `json:"user_id"`
		CreatedAt time.Time `json:"created_at"`
	}

	response := bookmarkResponse{
		ID:        bookmark.ID,
		PostID:    bookmark.PostID,
		UserID:    bookmark.UserID,
		CreatedAt: bookmark.CreatedAt,
	}

	s.respondWithJSON(w, http.StatusCreated, response)
}

func (s *Server) handleDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromContext(r)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	postIDStr := r.PathValue("postId")
	postID, err := uuid.Parse(postIDStr)
	if err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid post ID format")
		return
	}

	rowsAffected, err := s.db.DeleteBookmark(context.Background(), database.DeleteBookmarkParams{
		UserID: user.ID,
		PostID: postID,
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to delete bookmark")
		return
	}

	if rowsAffected == 0 {
		s.respondWithError(w, http.StatusNotFound, "Bookmark not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
