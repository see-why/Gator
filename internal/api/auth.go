package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gator/internal/database"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// User context key for storing authenticated user
type contextKey string

const userContextKey contextKey = "user"

// AuthenticatedUser represents a user in the request context
type AuthenticatedUser struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// generateAPIKey generates a secure random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// requireAuth is middleware that requires API key authentication
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.respondWithError(w, http.StatusUnauthorized, "API key required")
			return
		}

		// Extract API key (expect format: "ApiKey <key>")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "ApiKey" {
			s.respondWithError(w, http.StatusUnauthorized, "Invalid authorization format. Use: ApiKey <your-api-key>")
			return
		}

		apiKey := parts[1]
		if apiKey == "" {
			s.respondWithError(w, http.StatusUnauthorized, "API key cannot be empty")
			return
		}

		// Look up user by API key
		user, err := s.db.GetUserByAPIKey(context.Background(), sql.NullString{
			String: apiKey,
			Valid:  true,
		})
		if err != nil {
			s.respondWithError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		// Add user to request context
		authUser := AuthenticatedUser{
			ID:   user.ID,
			Name: user.Name,
		}
		ctx := context.WithValue(r.Context(), userContextKey, authUser)
		next(w, r.WithContext(ctx))
	}
}

// getUserFromContext extracts the authenticated user from request context
func getUserFromContext(r *http.Request) (AuthenticatedUser, error) {
	user, ok := r.Context().Value(userContextKey).(AuthenticatedUser)
	if !ok {
		return AuthenticatedUser{}, fmt.Errorf("user not found in context")
	}
	return user, nil
}

// Auth handlers
type registerRequest struct {
	Name string `json:"name"`
}

type registerResponse struct {
	User   AuthenticatedUser `json:"user"`
	APIKey string            `json:"api_key"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Name == "" {
		s.respondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	// Create user
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      req.Name,
		ApiKey:    sql.NullString{String: apiKey, Valid: true},
	})
	if err != nil {
		s.respondWithError(w, http.StatusConflict, "User already exists")
		return
	}

	response := registerResponse{
		User: AuthenticatedUser{
			ID:   user.ID,
			Name: user.Name,
		},
		APIKey: apiKey,
	}

	s.respondWithJSON(w, http.StatusCreated, response)
}

type loginRequest struct {
	Name string `json:"name"`
}

type loginResponse struct {
	User   AuthenticatedUser `json:"user"`
	APIKey string            `json:"api_key"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondWithError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Name == "" {
		s.respondWithError(w, http.StatusBadRequest, "Name is required")
		return
	}

	// Get user by name
	user, err := s.db.GetUser(context.Background(), req.Name)
	if err != nil {
		s.respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate new API key
	apiKey, err := generateAPIKey()
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	// Update user's API key
	err = s.db.UpdateUserAPIKey(context.Background(), database.UpdateUserAPIKeyParams{
		ID:     user.ID,
		ApiKey: sql.NullString{String: apiKey, Valid: true},
	})
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to update API key")
		return
	}

	response := loginResponse{
		User: AuthenticatedUser{
			ID:   user.ID,
			Name: user.Name,
		},
		APIKey: apiKey,
	}

	s.respondWithJSON(w, http.StatusOK, response)
}
