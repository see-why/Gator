package api

import (
	"html/template"
	"net/http"
)

const apiDocTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Gator RSS Reader API Documentation</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        h1, h2, h3 { color: #333; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        .endpoint { margin: 20px 0; padding: 15px; border-left: 4px solid #007cba; background: #f9f9f9; }
        .method { font-weight: bold; color: #007cba; }
        .auth { color: #d73527; font-weight: bold; }
    </style>
</head>
<body>
    <h1>Gator RSS Reader API Documentation</h1>
    
    <h2>Authentication</h2>
    <p>Most endpoints require authentication using an API key. Include your API key in the Authorization header:</p>
    <pre>Authorization: ApiKey &lt;your-api-key&gt;</pre>
    
    <h2>Endpoints</h2>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /health</h3>
        <p>Health check endpoint</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">POST</span> /api/auth/register</h3>
        <p>Register a new user</p>
        <pre>{
  "name": "username"
}</pre>
        <p>Returns user info and API key</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">POST</span> /api/auth/login</h3>
        <p>Login and get a new API key</p>
        <pre>{
  "name": "username"
}</pre>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/users <span class="auth">🔒 Auth Required</span></h3>
        <p>Get all users</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/users/me <span class="auth">🔒 Auth Required</span></h3>
        <p>Get current user info</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/feeds</h3>
        <p>Get all feeds</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">POST</span> /api/feeds <span class="auth">🔒 Auth Required</span></h3>
        <p>Create a new feed</p>
        <pre>{
  "name": "Feed Name",
  "url": "https://example.com/feed.xml"
}</pre>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/feed-follows <span class="auth">🔒 Auth Required</span></h3>
        <p>Get feeds you're following</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">POST</span> /api/feed-follows <span class="auth">🔒 Auth Required</span></h3>
        <p>Follow a feed</p>
        <pre>{
  "feed_url": "https://example.com/feed.xml"
}</pre>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">DELETE</span> /api/feed-follows <span class="auth">🔒 Auth Required</span></h3>
        <p>Unfollow a feed</p>
        <pre>{
  "feed_url": "https://example.com/feed.xml"
}</pre>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/posts <span class="auth">🔒 Auth Required</span></h3>
        <p>Get posts from feeds you follow</p>
        <p>Query parameters: <code>page</code> (default: 1), <code>limit</code> (default: 10, max: 100)</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/posts/search <span class="auth">🔒 Auth Required</span></h3>
        <p>Search posts</p>
        <p>Query parameters: <code>q</code> (required), <code>page</code> (default: 1), <code>limit</code> (default: 10, max: 100)</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> /api/bookmarks <span class="auth">🔒 Auth Required</span></h3>
        <p>Get your bookmarked posts</p>
        <p>Query parameters: <code>page</code> (default: 1), <code>limit</code> (default: 10, max: 100)</p>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">POST</span> /api/bookmarks <span class="auth">🔒 Auth Required</span></h3>
        <p>Bookmark a post</p>
        <pre>{
  "post_id": "uuid-of-post"
}</pre>
    </div>
    
    <div class="endpoint">
        <h3><span class="method">DELETE</span> /api/bookmarks/{postId} <span class="auth">🔒 Auth Required</span></h3>
        <p>Remove a bookmark</p>
    </div>
    
    <h2>Example Usage</h2>
    <pre># Register a new user
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name": "alice"}'

# Use the API key from registration response
export API_KEY="your-api-key-here"

# Create a feed
curl -X POST http://localhost:8080/api/feeds \
  -H "Authorization: ApiKey $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "Example Blog", "url": "https://example.com/feed.xml"}'

# Get posts
curl -H "Authorization: ApiKey $API_KEY" \
  "http://localhost:8080/api/posts?page=1&limit=5"

# Search posts
curl -H "Authorization: ApiKey $API_KEY" \
  "http://localhost:8080/api/posts/search?q=golang&page=1&limit=5"</pre>
</body>
</html>
`

func (s *Server) handleDocs(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("docs").Parse(apiDocTemplate)
	if err != nil {
		s.respondWithError(w, http.StatusInternalServerError, "Failed to render documentation")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	tmpl.Execute(w, nil)
}
