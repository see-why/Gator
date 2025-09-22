package tui

import (
	"context"
	"database/sql"
	"fmt"
	"gator/internal/database"
	"html"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/pkg/browser"
)

const (
	postsPerPage = 10
)

// PostItem represents a post in the TUI
type PostItem struct {
	ID          string
	Title       string
	URL         string
	FeedName    string
	Description string
	PublishedAt time.Time
	HasDate     bool
}

// Model represents the TUI state
type Model struct {
	db           *database.Queries
	userID       uuid.UUID
	posts        []PostItem
	currentPage  int
	cursor       int
	viewingPost  bool
	selectedPost PostItem
	loading      bool
	err          error
	width        int
	height       int
	searchMode   bool
	searchQuery  string
	isSearching  bool
}

type postsLoadedMsg struct {
	posts []PostItem
	err   error
}

// NewModel creates a new TUI model
func NewModel(db *database.Queries, userID uuid.UUID) Model {
	return Model{
		db:          db,
		userID:      userID,
		currentPage: 1,
		loading:     true,
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return m.loadPosts()
}

// Update handles TUI events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			if m.searchMode {
				m.searchMode = false
				m.searchQuery = ""
				m.isSearching = false
				return m, nil
			}
			if m.viewingPost {
				m.viewingPost = false
				return m, nil
			}
			return m, tea.Quit

		case "/":
			if !m.viewingPost {
				m.searchMode = true
				m.searchQuery = ""
				return m, nil
			}

		case "enter":
			if m.searchMode {
				m.searchMode = false
				m.isSearching = true
				m.loading = true
				m.cursor = 0
				m.currentPage = 1
				return m, m.searchPosts()
			}
			if !m.viewingPost && len(m.posts) > 0 {
				m.selectedPost = m.posts[m.cursor]
				m.viewingPost = true
				return m, nil
			}

		case "c":
			if !m.viewingPost && !m.searchMode {
				// Clear search and go back to browse mode
				m.searchQuery = ""
				m.isSearching = false
				m.loading = true
				m.cursor = 0
				m.currentPage = 1
				return m, m.loadPosts()
			}

		case "o":
			if m.viewingPost {
				// Open in browser
				browser.OpenURL(m.selectedPost.URL)
				return m, nil
			}

		case "up", "k":
			if !m.viewingPost && !m.searchMode && m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if !m.viewingPost && !m.searchMode && m.cursor < len(m.posts)-1 {
				m.cursor++
			}

		case "left", "h":
			if !m.viewingPost && !m.searchMode && m.currentPage > 1 {
				m.currentPage--
				m.cursor = 0
				m.loading = true
				if m.isSearching {
					return m, m.searchPosts()
				}
				return m, m.loadPosts()
			}

		case "right", "l":
			if !m.viewingPost && !m.searchMode {
				m.currentPage++
				m.cursor = 0
				m.loading = true
				if m.isSearching {
					return m, m.searchPosts()
				}
				return m, m.loadPosts()
			}

		case "backspace":
			if m.searchMode && len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			}

		default:
			if m.searchMode && len(msg.String()) == 1 {
				m.searchQuery += msg.String()
			}
		}

	case postsLoadedMsg:
		m.loading = false
		m.posts = msg.posts
		m.err = msg.err
		if m.cursor >= len(m.posts) {
			m.cursor = 0
		}
		return m, nil
	}

	return m, nil
}

// View renders the TUI
func (m Model) View() string {
	if m.loading {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("69")).
			Render("Loading posts...")
	}

	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(fmt.Sprintf("Error: %v", m.err))
	}

	if m.viewingPost {
		return m.renderPostView()
	}

	return m.renderPostList()
}

func (m Model) renderPostList() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("62")).
		Padding(0, 1)

	headerText := fmt.Sprintf("📰 Gator Posts - Page %d", m.currentPage)
	if m.isSearching && m.searchQuery != "" {
		headerText = fmt.Sprintf("🔍 Search: \"%s\" - Page %d", m.searchQuery, m.currentPage)
	}

	b.WriteString(headerStyle.Render(headerText))
	b.WriteString("\n\n")

	// Search input if in search mode
	if m.searchMode {
		searchStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

		searchText := fmt.Sprintf("Search: %s", m.searchQuery)
		if len(m.searchQuery) == 0 {
			searchText = "Search: (type to search, Enter to confirm, Esc to cancel)"
		}
		b.WriteString(searchStyle.Render(searchText))
		b.WriteString("\n\n")
		return b.String()
	}

	if len(m.posts) == 0 {
		noPostsText := "No posts found. Try following some feeds first!"
		if m.isSearching {
			noPostsText = fmt.Sprintf("No posts found matching \"%s\"", m.searchQuery)
		}
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Render(noPostsText))
		b.WriteString("\n\n")
	} else {
		// Posts list
		for i, post := range m.posts {
			style := lipgloss.NewStyle().Padding(0, 2)

			if i == m.cursor {
				style = style.
					Background(lipgloss.Color("62")).
					Foreground(lipgloss.Color("230")).
					Bold(true)
			}

			// Format the post
			postContent := fmt.Sprintf("▶ %s", truncate(post.Title, 60))
			if post.FeedName != "" {
				postContent += fmt.Sprintf(" [%s]", post.FeedName)
			}

			b.WriteString(style.Render(postContent))
			b.WriteString("\n")

			// Show description if available and not selected
			if i != m.cursor && post.Description != "" {
				descStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("244")).
					Padding(0, 4)
				b.WriteString(descStyle.Render(truncate(cleanHTML(post.Description), 80)))
				b.WriteString("\n")
			}
		}
	}

	// Controls
	b.WriteString("\n")
	controlsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	controls := "Navigate: ↑/k ↓/j  Pages: ←/h →/l  Select: Enter  Search: /  Clear: c  Quit: q"
	b.WriteString(controlsStyle.Render(controls))

	return b.String()
}

func (m Model) renderPostView() string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39"))

	b.WriteString(headerStyle.Render("📖 Post Details"))
	b.WriteString("\n\n")

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("208")).
		Width(m.width - 4)

	b.WriteString(titleStyle.Render(m.selectedPost.Title))
	b.WriteString("\n\n")

	// Metadata
	metaStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Italic(true)

	if m.selectedPost.FeedName != "" {
		b.WriteString(metaStyle.Render(fmt.Sprintf("Feed: %s", m.selectedPost.FeedName)))
		b.WriteString("\n")
	}

	if m.selectedPost.HasDate {
		b.WriteString(metaStyle.Render(fmt.Sprintf("Published: %s", m.selectedPost.PublishedAt.Format("2006-01-02 15:04:05"))))
		b.WriteString("\n")
	}

	b.WriteString(metaStyle.Render(fmt.Sprintf("URL: %s", m.selectedPost.URL)))
	b.WriteString("\n\n")

	// Description
	if m.selectedPost.Description != "" {
		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(m.width - 4)

		cleanDesc := cleanHTML(m.selectedPost.Description)
		b.WriteString(descStyle.Render(cleanDesc))
		b.WriteString("\n\n")
	}

	// Controls
	controlsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	controls := "Open in browser: o  Back: Esc  Quit: q/Ctrl+C"
	b.WriteString(controlsStyle.Render(controls))

	return b.String()
}

func (m Model) loadPosts() tea.Cmd {
	return func() tea.Msg {
		offset := int32((m.currentPage - 1) * postsPerPage)

		posts, err := m.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
			UserID: m.userID,
			Limit:  postsPerPage,
			Offset: offset,
		})

		if err != nil {
			return postsLoadedMsg{err: err}
		}

		// Convert to PostItem
		items := make([]PostItem, len(posts))
		for i, post := range posts {
			items[i] = PostItem{
				ID:          post.ID.String(),
				Title:       post.Title,
				URL:         post.Url,
				FeedName:    post.FeedName,
				Description: post.Description.String,
				HasDate:     post.PublishedAt.Valid,
			}
			if post.PublishedAt.Valid {
				items[i].PublishedAt = post.PublishedAt.Time
			}
		}

		return postsLoadedMsg{posts: items}
	}
}

func (m Model) searchPosts() tea.Cmd {
	return func() tea.Msg {
		offset := int32((m.currentPage - 1) * postsPerPage)

		posts, err := m.db.SearchPostsForUser(context.Background(), database.SearchPostsForUserParams{
			UserID:  m.userID,
			Column2: sql.NullString{String: m.searchQuery, Valid: true},
			Limit:   postsPerPage,
			Offset:  offset,
		})

		if err != nil {
			return postsLoadedMsg{err: err}
		}

		// Convert to PostItem
		items := make([]PostItem, len(posts))
		for i, post := range posts {
			items[i] = PostItem{
				ID:          post.ID.String(),
				Title:       post.Title,
				URL:         post.Url,
				FeedName:    post.FeedName,
				Description: post.Description.String,
				HasDate:     post.PublishedAt.Valid,
			}
			if post.PublishedAt.Valid {
				items[i].PublishedAt = post.PublishedAt.Time
			}
		}

		return postsLoadedMsg{posts: items}
	}
}

// Helper functions
func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

func cleanHTML(s string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")

	// Decode HTML entities
	s = html.UnescapeString(s)

	// Clean up whitespace
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	return s
}

// RunTUI starts the TUI application
func RunTUI(db *database.Queries, userID uuid.UUID) error {
	model := NewModel(db, userID)

	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err := program.Run()
	return err
}
