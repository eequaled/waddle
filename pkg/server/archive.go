package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type ArchiveGroup struct {
	Name  string   `json:"name"`
	Items []string `json:"items"` // List of archived items in this group
}

// handleArchives handles GET /api/archives and POST /api/archives
func (s *Server) handleArchives(w http.ResponseWriter, r *http.Request) {
	archivesDir := filepath.Join(s.rootDir, "archives")
	os.MkdirAll(archivesDir, 0755)

	if r.Method == "GET" {
		entries, err := os.ReadDir(archivesDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var groups []ArchiveGroup
		for _, e := range entries {
			if e.IsDir() {
				// List items in this group
				groupDir := filepath.Join(archivesDir, e.Name())
				items, _ := os.ReadDir(groupDir)
				var itemNames []string
				for _, item := range items {
					itemNames = append(itemNames, item.Name())
				}
				groups = append(groups, ArchiveGroup{
					Name:  e.Name(),
					Items: itemNames,
				})
			}
		}
		json.NewEncoder(w).Encode(groups)
		return
	}

	if r.Method == "POST" {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Create Group Directory
		groupPath := filepath.Join(archivesDir, req.Name)
		if err := os.MkdirAll(groupPath, 0755); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleArchiveMove handles POST /api/archives/move
func (s *Server) handleArchiveMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID   string `json:"sessionId"` // Format: YYYY-MM-DD
		AppName     string `json:"appName"`   // Optional: if moving specific app
		TargetGroup string `json:"targetGroup"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sourcePath := ""
	destName := ""

	if req.AppName != "" {
		// Moving specific app session
		sourcePath = filepath.Join(s.rootDir, req.SessionID, req.AppName)
		destName = fmt.Sprintf("%s_%s", req.SessionID, req.AppName)
	} else {
		// Moving entire day
		sourcePath = filepath.Join(s.rootDir, req.SessionID)
		destName = req.SessionID
	}

	destPath := filepath.Join(s.rootDir, "archives", req.TargetGroup, destName)

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		http.Error(w, "Source not found", http.StatusNotFound)
		return
	}

	// Move (Rename)
	// Note: Rename might fail across volumes, but usually fine here
	if err := os.Rename(sourcePath, destPath); err != nil {
		// Fallback: Copy and Delete (not implemented for brevity, assuming same volume)
		http.Error(w, fmt.Sprintf("Failed to move: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "moved"})
}
