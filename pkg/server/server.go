package server

import (
	"encoding/json"
	"fmt"
	"ideathon/pkg/storage"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Server struct {
	rootDir       string
	port          string
	isPaused      *atomic.Bool
	storageEngine *storage.StorageEngine
}

func NewServer(rootDir string, port string, isPaused *atomic.Bool, storageEngine *storage.StorageEngine) *Server {
	return &Server{
		rootDir:       rootDir,
		port:          port,
		isPaused:      isPaused,
		storageEngine: storageEngine,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	// Enable CORS
	cors := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				return
			}
			h(w, r)
		}
	}

	// API Endpoints
	mux.HandleFunc("/api/sessions", cors(s.handleSessions))
	mux.HandleFunc("/api/sessions/", cors(s.handleAppDetails)) // Wildcard for dates

	// New search endpoints
	mux.HandleFunc("/api/search/fulltext", cors(s.handleFullTextSearch))
	mux.HandleFunc("/api/search/semantic", cors(s.handleSemanticSearch))

	// Status Endpoint
	mux.HandleFunc("/api/status", cors(s.handleStatus))

	// Health Endpoint
	mux.HandleFunc("/api/health", cors(s.handleHealth))

	// Blacklist Endpoint
	mux.HandleFunc("/api/blacklist", cors(s.handleBlacklist))

	// Chat Endpoints
	mux.HandleFunc("/api/chat", cors(s.handleChat))

	// Archive Endpoints
	mux.HandleFunc("/api/archives", cors(s.handleArchives))
	mux.HandleFunc("/api/archives/move", cors(s.handleArchiveMove))

	// Notification Endpoints
	mux.HandleFunc("/api/notifications", cors(s.handleNotifications))
	mux.HandleFunc("/api/notifications/read", cors(s.handleNotificationsRead))

	// Profile Endpoints
	mux.HandleFunc("/api/profile/images", cors(s.handleProfileImages))
	mux.HandleFunc("/api/profile/upload", cors(s.handleProfileUpload))
	mux.HandleFunc("/api/profile/delete", cors(s.handleProfileDelete))

	// Static Files (Images)
	fileServer := http.FileServer(http.Dir(s.rootDir))
	mux.Handle("/images/", http.StripPrefix("/images/", fileServer))

	fmt.Printf("Starting API Server on port %s...\n", s.port)
	go http.ListenAndServe(":"+s.port, mux)
}

// GET /api/sessions -> Returns list of dates [ "2023-10-27", ... ]
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Use StorageEngine to list sessions
	sessions, _, err := s.storageEngine.ListSessions(1, 1000) // Get all sessions
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var dates []string
	for _, session := range sessions {
		dates = append(dates, session.Date)
	}

	// Sort reverse (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	json.NewEncoder(w).Encode(dates)
}

// GET /api/search/fulltext -> Full-text search
func (s *Server) handleFullTextSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 50
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	results, err := s.storageEngine.FullTextSearch(query, page, pageSize)
	if err != nil {
		// Return empty array instead of error for compatibility
		json.NewEncoder(w).Encode([]storage.SearchResult{})
		return
	}

	json.NewEncoder(w).Encode(results)
}

// GET /api/search/semantic -> Semantic search
func (s *Server) handleSemanticSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	topK := 10
	if k := r.URL.Query().Get("topK"); k != "" {
		if parsed, err := strconv.Atoi(k); err == nil && parsed > 0 && parsed <= 100 {
			topK = parsed
		}
	}

	var dateRange *storage.DateRange
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")
	if startDate != "" || endDate != "" {
		dateRange = &storage.DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		}
	}

	results, err := s.storageEngine.SemanticSearch(query, topK, dateRange)
	if err != nil {
		// Return empty array instead of error for compatibility
		json.NewEncoder(w).Encode([]storage.SearchResult{})
		return
	}

	json.NewEncoder(w).Encode(results)
}

// GET /api/health -> Returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health, err := s.storageEngine.HealthCheck()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(health)
}

// GET /api/status -> Returns { "paused": bool }
// POST /api/status -> Body { "paused": bool } -> Updates status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		json.NewEncoder(w).Encode(map[string]bool{
			"paused": s.isPaused.Load(),
		})
		return
	}

	if r.Method == "POST" {
		var body struct {
			Paused bool `json:"paused"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.isPaused.Store(body.Paused)
		json.NewEncoder(w).Encode(map[string]bool{
			"paused": s.isPaused.Load(),
		})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// GET /api/blacklist -> Returns [ "app.exe", ... ]
// POST /api/blacklist -> Body [ "app.exe", ... ] -> Writes to file
func (s *Server) handleBlacklist(w http.ResponseWriter, r *http.Request) {
	blacklistPath := filepath.Join(s.rootDir, "blacklist.txt")

	if r.Method == "GET" {
		content, err := os.ReadFile(blacklistPath)
		if err != nil {
			if os.IsNotExist(err) {
				json.NewEncoder(w).Encode([]string{})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lines := strings.Split(string(content), "\n")
		var apps []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				apps = append(apps, trimmed)
			}
		}
		json.NewEncoder(w).Encode(apps)
		return
	}

	if r.Method == "POST" {
		var apps []string
		if err := json.NewDecoder(r.Body).Decode(&apps); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Write to file
		data := strings.Join(apps, "\n")
		if err := os.WriteFile(blacklistPath, []byte(data), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(apps)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// GET /api/sessions/{date} -> Returns list of apps
// PUT /api/sessions/{date} -> Updates session metadata
// DELETE /api/sessions/{date} -> Deletes session
// GET /api/sessions/{date}/{app} -> Returns details (images, text)
func (s *Server) handleAppDetails(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")

	// Handle PUT for session update
	if r.Method == "PUT" && len(parts) == 1 && parts[0] != "" {
		date := parts[0]
		s.handleSessionUpdate(w, r, date)
		return
	}

	// Handle DELETE for session deletion
	if r.Method == "DELETE" && len(parts) == 1 && parts[0] != "" {
		date := parts[0]
		s.handleSessionDelete(w, r, date)
		return
	}

	if len(parts) == 1 && parts[0] != "" {
		// List Apps for Date - use StorageEngine to get activity blocks
		date := parts[0]
		
		// Verify session exists
		_, err := s.storageEngine.GetSession(date)
		if err != nil {
			if storage.IsNotFound(err) {
				http.Error(w, "Date not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// For now, return empty list since we need to implement app listing
		// This would require querying the database for app activities
		// TODO: Implement proper app listing from database
		json.NewEncoder(w).Encode([]string{})
		return
	}

	if len(parts) == 2 && parts[1] == "metadata" {
		// Get Session Metadata using StorageEngine
		date := parts[0]
		session, err := s.storageEngine.GetSession(date)
		if err != nil {
			if storage.IsNotFound(err) {
				// Return empty/default metadata
				json.NewEncoder(w).Encode(SessionMetadata{})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert storage.Session to API SessionMetadata format
		metadata := SessionMetadata{
			CustomTitle:     session.CustomTitle,
			CustomSummary:   session.CustomSummary,
			OriginalSummary: session.OriginalSummary,
			ManualNotes:     []ManualNote{}, // Initialize as empty slice
		}

		json.NewEncoder(w).Encode(metadata)
		return
	}

	if len(parts) >= 2 {
		// Get App Details
		date := parts[0]
		app := parts[1]

		// Verify session exists
		_, err := s.storageEngine.GetSession(date)
		if err != nil {
			if storage.IsNotFound(err) {
				http.Error(w, "Session not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Check if requesting blocks
		if len(parts) == 3 && parts[2] == "blocks" {
			blocks, err := s.storageEngine.GetActivityBlocks(date, app)
			if err != nil {
				// Return empty list if no blocks yet
				json.NewEncoder(w).Encode([]interface{}{})
				return
			}

			// Convert storage.ActivityBlock to API format
			type BlockData struct {
				ID           string `json:"id"`
				StartTime    string `json:"startTime"`
				EndTime      string `json:"endTime"`
				MicroSummary string `json:"microSummary"`
				OCRText      string `json:"ocrText"`
			}
			var apiBlocks []BlockData

			for _, block := range blocks {
				apiBlocks = append(apiBlocks, BlockData{
					ID:           block.BlockID,
					StartTime:    block.StartTime.Format(time.RFC3339),
					EndTime:      block.EndTime.Format(time.RFC3339),
					MicroSummary: block.MicroSummary,
					OCRText:      block.OCRText,
				})
			}
			json.NewEncoder(w).Encode(apiBlocks)
			return
		}

		// List files - for now, return empty since files are handled by filesystem
		// TODO: Implement file listing through StorageEngine
		type FileInfo struct {
			Name string `json:"name"`
			Type string `json:"type"` // "image" or "text"
			Url  string `json:"url"`
		}
		var files []FileInfo
		json.NewEncoder(w).Encode(files)
		return
	}
}

// ManualNote represents a user-added note
type ManualNote struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// SessionMetadata stores user customizations for a session
type SessionMetadata struct {
	CustomTitle     string       `json:"customTitle,omitempty"`
	CustomSummary   string       `json:"customSummary,omitempty"`
	OriginalSummary string       `json:"originalSummary,omitempty"`
	ManualNotes     []ManualNote `json:"manualNotes"`
}

// PUT /api/sessions/{date} -> Updates session metadata
func (s *Server) handleSessionUpdate(w http.ResponseWriter, r *http.Request, date string) {
	// Get existing session
	session, err := s.storageEngine.GetSession(date)
	if err != nil {
		if storage.IsNotFound(err) {
			// Create new session if it doesn't exist
			session, err = s.storageEngine.CreateSession(date)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var metadata SessionMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update session with new metadata
	session.CustomTitle = metadata.CustomTitle
	session.CustomSummary = metadata.CustomSummary
	session.OriginalSummary = metadata.OriginalSummary

	if err := s.storageEngine.UpdateSession(session); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DELETE /api/sessions/{date} -> Deletes session
func (s *Server) handleSessionDelete(w http.ResponseWriter, _ *http.Request, date string) {
	if err := s.storageEngine.DeleteSession(date); err != nil {
		if storage.IsNotFound(err) {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// Notification represents a user notification
type Notification struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"` // status, insight, processing
	Title      string            `json:"title"`
	Message    string            `json:"message"`
	Timestamp  string            `json:"timestamp"`
	Read       bool              `json:"read"`
	SessionRef string            `json:"sessionRef,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// GET /api/notifications -> Returns list of notifications
// POST /api/notifications -> Creates a new notification
func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Use StorageEngine to get notifications
		notifications, err := s.storageEngine.GetNotifications(100) // Get last 100
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert storage.Notification to API Notification format
		var apiNotifications []Notification
		for _, notif := range notifications {
			apiNotif := Notification{
				ID:         notif.ID,
				Type:       notif.Type,
				Title:      notif.Title,
				Message:    notif.Message,
				Timestamp:  notif.Timestamp.Format(time.RFC3339),
				Read:       notif.Read,
				SessionRef: notif.SessionRef,
			}
			
			// Parse metadata JSON string to map
			if notif.Metadata != "" {
				var metadata map[string]string
				if err := json.Unmarshal([]byte(notif.Metadata), &metadata); err == nil {
					apiNotif.Metadata = metadata
				}
			}
			
			apiNotifications = append(apiNotifications, apiNotif)
		}

		json.NewEncoder(w).Encode(apiNotifications)
		return
	}

	if r.Method == "POST" {
		var newNotif Notification
		if err := json.NewDecoder(r.Body).Decode(&newNotif); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Convert API Notification to storage.Notification
		storageNotif := &storage.Notification{
			ID:         fmt.Sprintf("notif-%d", time.Now().UnixNano()),
			Type:       newNotif.Type,
			Title:      newNotif.Title,
			Message:    newNotif.Message,
			Timestamp:  time.Now(),
			Read:       false,
			SessionRef: newNotif.SessionRef,
		}
		
		// Convert metadata map to JSON string
		if newNotif.Metadata != nil {
			metadataBytes, err := json.Marshal(newNotif.Metadata)
			if err == nil {
				storageNotif.Metadata = string(metadataBytes)
			}
		}

		if err := s.storageEngine.AddNotification(storageNotif); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return the created notification in API format
		newNotif.ID = storageNotif.ID
		newNotif.Timestamp = storageNotif.Timestamp.Format(time.RFC3339)
		newNotif.Read = false

		json.NewEncoder(w).Encode(newNotif)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// POST /api/notifications/read -> Marks notifications as read
func (s *Server) handleNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Use StorageEngine to mark notifications as read
	if err := s.storageEngine.MarkNotificationsRead(body.IDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GET /api/profile/images -> Returns list of image filenames
func (s *Server) handleProfileImages(w http.ResponseWriter, r *http.Request) {
	profileDir := filepath.Join(s.rootDir, "profile")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		// Create if not exists
		os.Mkdir(profileDir, 0755)
		json.NewEncoder(w).Encode([]string{})
		return
	}

	entries, err := os.ReadDir(profileDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var images []string
	for _, e := range entries {
		if !e.IsDir() && (strings.HasSuffix(e.Name(), ".png") || strings.HasSuffix(e.Name(), ".jpg") || strings.HasSuffix(e.Name(), ".jpeg")) {
			images = append(images, e.Name())
		}
	}

	// Debug logging
	fmt.Printf("[DEBUG] Profile images found: %v\n", images)

	json.NewEncoder(w).Encode(images)
}

// POST /api/profile/upload -> Uploads a new profile image
func (s *Server) handleProfileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit upload size to 10MB
	r.ParseMultipartForm(10 << 20)

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	contentType := http.DetectContentType(buffer)
	if contentType != "image/jpeg" && contentType != "image/png" {
		http.Error(w, "Invalid file type. Only JPEG and PNG are allowed.", http.StatusBadRequest)
		return
	}

	// Seek back to start of file
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, "Error processing file", http.StatusInternalServerError)
		return
	}

	profileDir := filepath.Join(s.rootDir, "profile")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		os.Mkdir(profileDir, 0755)
	}

	// Create a safe filename (extension based on detected content type, not user input)
	ext := ".png"
	if contentType == "image/jpeg" {
		ext = ".jpg"
	}

	filename := fmt.Sprintf("upload-%d%s", time.Now().UnixNano(), ext)
	dstPath := filepath.Join(profileDir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"filename": filename,
		"url":      fmt.Sprintf("http://localhost:%s/images/profile/%s", s.port, filename),
	})
}

// DELETE /api/profile/delete -> Deletes a profile image
func (s *Server) handleProfileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Prevent deleting default images
	if body.Filename == "default_1.png" || body.Filename == "default_2.png" {
		http.Error(w, "Cannot delete default images", http.StatusForbidden)
		return
	}

	profileDir := filepath.Join(s.rootDir, "profile")
	filePath := filepath.Join(profileDir, body.Filename)

	// Security check - ensure the file is within profile directory
	if !strings.HasPrefix(filePath, profileDir) {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
