package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

type Server struct {
	rootDir  string
	port     string
	isPaused *atomic.Bool
}

func NewServer(rootDir string, port string, isPaused *atomic.Bool) *Server {
	return &Server{
		rootDir:  rootDir,
		port:     port,
		isPaused: isPaused,
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

	// Status Endpoint
	mux.HandleFunc("/api/status", cors(s.handleStatus))

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
	entries, err := os.ReadDir(s.rootDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var dates []string
	for _, e := range entries {
		if e.IsDir() && e.Name() != "profile" {
			dates = append(dates, e.Name())
		}
	}
	// Sort reverse (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	json.NewEncoder(w).Encode(dates)
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
		// List Apps for Date
		date := parts[0]
		dateDir := filepath.Join(s.rootDir, date)
		entries, err := os.ReadDir(dateDir)
		if err != nil {
			http.Error(w, "Date not found", http.StatusNotFound)
			return
		}

		type AppEntry struct {
			Name    string
			ModTime int64
		}
		var appEntries []AppEntry

		for _, e := range entries {
			if e.IsDir() {
				info, err := e.Info()
				if err == nil {
					appEntries = append(appEntries, AppEntry{
						Name:    e.Name(),
						ModTime: info.ModTime().Unix(),
					})
				}
			}
		}

		// Sort by ModTime descending
		sort.Slice(appEntries, func(i, j int) bool {
			return appEntries[i].ModTime > appEntries[j].ModTime
		})

		var apps []string
		for _, e := range appEntries {
			apps = append(apps, e.Name)
		}

		json.NewEncoder(w).Encode(apps)
		return
	}

	if len(parts) >= 2 {
		// Get App Details
		date := parts[0]
		app := parts[1]
		appDir := filepath.Join(s.rootDir, date, app)

		if _, err := os.Stat(appDir); os.IsNotExist(err) {
			http.Error(w, "App not found", http.StatusNotFound)
			return
		}

		// Check if requesting blocks
		if len(parts) == 3 && parts[2] == "blocks" {
			blocksDir := filepath.Join(appDir, "blocks")
			entries, err := os.ReadDir(blocksDir)
			if err != nil {
				// Return empty list if no blocks yet
				json.NewEncoder(w).Encode([]interface{}{})
				return
			}

			type BlockData struct {
				ID           string `json:"id"`
				StartTime    string `json:"startTime"`
				EndTime      string `json:"endTime"`
				MicroSummary string `json:"microSummary"`
				OCRText      string `json:"ocrText"`
			}
			var blocks []BlockData

			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".json") {
					content, _ := os.ReadFile(filepath.Join(blocksDir, e.Name()))
					var block BlockData
					json.Unmarshal(content, &block)
					blocks = append(blocks, block)
				}
			}
			json.NewEncoder(w).Encode(blocks)
			return
		}

		// List files (Default behavior)
		entries, err := os.ReadDir(appDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type FileInfo struct {
			Name string `json:"name"`
			Type string `json:"type"` // "image" or "text"
			Url  string `json:"url"`
		}
		var files []FileInfo

		for _, e := range entries {
			if !e.IsDir() {
				fType := "unknown"
				if strings.HasSuffix(e.Name(), ".png") {
					fType = "image"
				} else if strings.HasSuffix(e.Name(), ".txt") {
					fType = "text"
				}

				url := fmt.Sprintf("http://localhost:%s/images/%s/%s/%s", s.port, date, app, e.Name())
				files = append(files, FileInfo{
					Name: e.Name(),
					Type: fType,
					Url:  url,
				})
			}
		}
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
	ManualNotes     []ManualNote `json:"manualNotes,omitempty"`
}

// PUT /api/sessions/{date} -> Updates session metadata
func (s *Server) handleSessionUpdate(w http.ResponseWriter, r *http.Request, date string) {
	dateDir := filepath.Join(s.rootDir, date)
	if _, err := os.Stat(dateDir); os.IsNotExist(err) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	var metadata SessionMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Save metadata to a JSON file in the session directory
	metadataPath := filepath.Join(dateDir, "metadata.json")
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// DELETE /api/sessions/{date} -> Deletes session
func (s *Server) handleSessionDelete(w http.ResponseWriter, _ *http.Request, date string) {
	dateDir := filepath.Join(s.rootDir, date)
	if _, err := os.Stat(dateDir); os.IsNotExist(err) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Remove the entire session directory
	if err := os.RemoveAll(dateDir); err != nil {
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
	notificationsPath := filepath.Join(s.rootDir, "notifications.json")

	if r.Method == "GET" {
		content, err := os.ReadFile(notificationsPath)
		if err != nil {
			if os.IsNotExist(err) {
				json.NewEncoder(w).Encode([]Notification{})
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var notifications []Notification
		if err := json.Unmarshal(content, &notifications); err != nil {
			json.NewEncoder(w).Encode([]Notification{})
			return
		}

		// Sort by timestamp descending (newest first)
		sort.Slice(notifications, func(i, j int) bool {
			return notifications[i].Timestamp > notifications[j].Timestamp
		})

		json.NewEncoder(w).Encode(notifications)
		return
	}

	if r.Method == "POST" {
		var newNotif Notification
		if err := json.NewDecoder(r.Body).Decode(&newNotif); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Generate ID and timestamp
		newNotif.ID = fmt.Sprintf("notif-%d", time.Now().UnixNano())
		newNotif.Timestamp = time.Now().Format(time.RFC3339)
		newNotif.Read = false

		// Load existing notifications
		var notifications []Notification
		content, err := os.ReadFile(notificationsPath)
		if err == nil {
			json.Unmarshal(content, &notifications)
		}

		// Add new notification at the beginning
		notifications = append([]Notification{newNotif}, notifications...)

		// Keep only last 100 notifications
		if len(notifications) > 100 {
			notifications = notifications[:100]
		}

		// Save
		data, _ := json.MarshalIndent(notifications, "", "  ")
		os.WriteFile(notificationsPath, data, 0644)

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

	notificationsPath := filepath.Join(s.rootDir, "notifications.json")

	// Load existing notifications
	var notifications []Notification
	content, err := os.ReadFile(notificationsPath)
	if err != nil {
		http.Error(w, "No notifications found", http.StatusNotFound)
		return
	}
	json.Unmarshal(content, &notifications)

	// Mark specified notifications as read
	idSet := make(map[string]bool)
	for _, id := range body.IDs {
		idSet[id] = true
	}

	for i := range notifications {
		if idSet[notifications[i].ID] {
			notifications[i].Read = true
		}
	}

	// Save
	data, _ := json.MarshalIndent(notifications, "", "  ")
	os.WriteFile(notificationsPath, data, 0644)

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

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	profileDir := filepath.Join(s.rootDir, "profile")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		os.Mkdir(profileDir, 0755)
	}

	// Create a safe filename
	filename := fmt.Sprintf("upload-%d%s", time.Now().UnixNano(), filepath.Ext(handler.Filename))
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
