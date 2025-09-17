package api

import (
	"encoding/json"
	"gator/internal/database"
	"log"
	"net/http"
	"time"
)

// Server holds the HTTP server and dependencies
type Server struct {
	db     *database.Queries
	router *http.ServeMux
	port   string
}

// NewServer creates a new HTTP server instance
func NewServer(db *database.Queries, port string) *Server {
	s := &Server{
		db:     db,
		router: http.NewServeMux(),
		port:   port,
	}
	s.setupRoutes()
	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on port %s", s.port)
	return http.ListenAndServe(":"+s.port, s.router)
}

// setupRoutes configures all the API endpoints
func (s *Server) setupRoutes() {
	// Health check and docs
	s.router.HandleFunc("GET /health", s.handleHealth)
	s.router.HandleFunc("GET /api/docs", s.handleDocs)

	// Authentication
	s.router.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.router.HandleFunc("POST /api/auth/login", s.handleLogin)

	// User endpoints (require authentication)
	s.router.HandleFunc("GET /api/users", s.requireAuth(s.handleGetUsers))
	s.router.HandleFunc("GET /api/users/me", s.requireAuth(s.handleGetCurrentUser))

	// Feed endpoints
	s.router.HandleFunc("GET /api/feeds", s.handleGetFeeds)
	s.router.HandleFunc("POST /api/feeds", s.requireAuth(s.handleCreateFeed))

	// Feed follow endpoints
	s.router.HandleFunc("GET /api/feed-follows", s.requireAuth(s.handleGetFeedFollows))
	s.router.HandleFunc("POST /api/feed-follows", s.requireAuth(s.handleCreateFeedFollow))
	s.router.HandleFunc("DELETE /api/feed-follows", s.requireAuth(s.handleDeleteFeedFollow))

	// Post endpoints
	s.router.HandleFunc("GET /api/posts", s.requireAuth(s.handleGetPosts))
	s.router.HandleFunc("GET /api/posts/search", s.requireAuth(s.handleSearchPosts))

	// Bookmark endpoints
	s.router.HandleFunc("GET /api/bookmarks", s.requireAuth(s.handleGetBookmarks))
	s.router.HandleFunc("POST /api/bookmarks", s.requireAuth(s.handleCreateBookmark))
	s.router.HandleFunc("DELETE /api/bookmarks/{postId}", s.requireAuth(s.handleDeleteBookmark))

	// Like endpoints
	s.router.HandleFunc("GET /api/likes", s.requireAuth(s.handleGetLikes))
	s.router.HandleFunc("POST /api/likes", s.requireAuth(s.handleCreateLike))
	s.router.HandleFunc("DELETE /api/likes/{postId}", s.requireAuth(s.handleDeleteLike))
}

// Response helpers
func (s *Server) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func (s *Server) respondWithError(w http.ResponseWriter, code int, message string) {
	type errorResponse struct {
		Error string `json:"error"`
	}
	s.respondWithJSON(w, code, errorResponse{Error: message})
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	type healthResponse struct {
		Status string `json:"status"`
		Time   string `json:"time"`
	}
	s.respondWithJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	})
}
