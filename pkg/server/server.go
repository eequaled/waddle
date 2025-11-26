package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Server struct {
	rootDir string
	port    string
}

func NewServer(rootDir string, port string) *Server {
	return &Server{
		rootDir: rootDir,
		port:    port,
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()

	// Enable CORS
	cors := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
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
		if e.IsDir() {
			dates = append(dates, e.Name())
		}
	}
	// Sort reverse (newest first)
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	json.NewEncoder(w).Encode(dates)
}

// GET /api/sessions/{date} -> Returns list of apps
// GET /api/sessions/{date}/{app} -> Returns details (images, text)
func (s *Server) handleAppDetails(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")

	if len(parts) == 1 && parts[0] != "" {
		// List Apps for Date
		date := parts[0]
		dateDir := filepath.Join(s.rootDir, date)
		entries, err := os.ReadDir(dateDir)
		if err != nil {
			http.Error(w, "Date not found", http.StatusNotFound)
			return
		}

		var apps []string
		for _, e := range entries {
			if e.IsDir() {
				apps = append(apps, e.Name())
			}
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

		// List files
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
