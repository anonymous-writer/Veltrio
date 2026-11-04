package app

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
	webassets "veltrio"
)

type Server struct {
	manager               *Manager
	assets                fs.FS
	version               string
	downloadDir           string
	maxConcurrent         int
	resumeOnStart         bool
	deleteFileOnRemove    bool
	logLevel              string
	ntfyEnabled           bool
	ntfyDebug             bool
	databaseDebug         bool
	ntfyURL               string
	ntfyPriority          string
	ntfyNotifyUnknownIPv4 bool
	mux                   *http.ServeMux
}

func New(stateFile, downloadDir, version string) (*Server, error) {
	return NewWithConfig(stateFile, downloadDir, version, 3, true, true, "info")
}
func NewWithConfig(stateFile, downloadDir, version string, maxConcurrent int, resumeOnStart, deleteFileOnRemove bool, logLevel string) (*Server, error) {
	assets := webassets.FS()
	m, err := NewManagerWithOptions(stateFile, downloadDir, maxConcurrent, resumeOnStart, deleteFileOnRemove)
	if err != nil {
		return nil, err
	}
	s := &Server{manager: m, assets: assets, mux: http.NewServeMux(), version: version, downloadDir: downloadDir, maxConcurrent: maxConcurrent, resumeOnStart: resumeOnStart, deleteFileOnRemove: deleteFileOnRemove, logLevel: logLevel}
	s.routes()
	return s, nil
}
func (s *Server) Handler() http.Handler { return s.mux }
func (s *Server) ConfigureNtfy(n Ntfy, databaseDebug, notifyUnknownIPv4 bool) {
	s.ntfyEnabled = n.Enabled && strings.TrimSpace(n.URL) != ""
	s.ntfyDebug = n.Debug
	s.databaseDebug = databaseDebug
	s.ntfyNotifyUnknownIPv4 = notifyUnknownIPv4
	s.ntfyURL = n.URL
	s.ntfyPriority = n.Priority
	if n.Enabled && strings.TrimSpace(n.URL) != "" {
		s.manager.SetNotifier(func(title, message string) error { return n.Notify(title, message) })
	}
}
func (s *Server) routes() {
	s.mux.HandleFunc("/api/downloads", s.downloads)
	s.mux.HandleFunc("/api/downloads/", s.downloadAction)
	s.mux.HandleFunc("/api/version", s.versionStatus)
	s.mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"status": "ok", "serverTime": time.Now().Format(time.RFC3339), "downloadDir": s.downloadDir, "maxConcurrent": s.maxConcurrent, "resumeOnStart": s.resumeOnStart, "deleteFileOnRemove": s.deleteFileOnRemove, "logLevel": s.logLevel, "ntfyEnabled": s.ntfyEnabled, "ntfyDebug": s.ntfyDebug, "databaseDebug": s.databaseDebug, "ntfyPriority": s.ntfyPriority, "ntfyNotifyUnknownIPv4": s.ntfyNotifyUnknownIPv4})
	})
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "name": "Veltrio"})
	})
	s.mux.HandleFunc("/", s.frontend)
}
func (s *Server) frontend(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "templates/index.html"
	}
	if strings.HasPrefix(name, "static/") {
		data, err := fs.ReadFile(s.assets, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if strings.HasSuffix(name, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		if strings.HasSuffix(name, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		}
		if strings.HasSuffix(name, ".svg") {
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		_, _ = w.Write(data)
		return
	}
	if name == "app.css" {
		name = "static/app.css"
	}
	if name == "app.js" {
		name = "static/app.js"
	}
	if name == "logo.svg" {
		name = "static/logo.svg"
	}
	if name == "templates/index.html" {
		data, err := fs.ReadFile(s.assets, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
		return
	}
	if _, err := s.assets.Open(name); err != nil {
		http.NotFound(w, r)
		return
	}
	http.FileServer(http.FS(s.assets)).ServeHTTP(w, r)
}
func (s *Server) versionStatus(w http.ResponseWriter, r *http.Request) {
	type status struct {
		Version       string `json:"version"`
		LatestVersion string `json:"latestVersion,omitempty"`
		Channel       string `json:"channel"`
		Repository    string `json:"repository"`
		ReleaseURL    string `json:"releaseUrl,omitempty"`
		UpdatedAt     string `json:"updatedAt,omitempty"`
		RunURL        string `json:"runUrl,omitempty"`
		Message       string `json:"message,omitempty"`
	}
	result := status{Version: s.version, Channel: "Release", Repository: "https://github.com/anonymous-writer/Veltrio"}
	client := &http.Client{Timeout: 5 * time.Second}
	fetch := func(endpoint string, target any) error {
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "Veltrio")
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("GitHub HTTP %s", resp.Status)
		}
		return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(target)
	}
	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := fetch("https://api.github.com/repos/anonymous-writer/Veltrio/releases/latest", &release); err == nil && release.TagName != "" {
		result.LatestVersion = release.TagName
		result.ReleaseURL = release.HTMLURL
	} else {
		result.Message = "GitHub-Status momentan nicht verfügbar"
	}
	var runs struct {
		WorkflowRuns []struct {
			HTMLURL    string `json:"html_url"`
			UpdatedAt  string `json:"updated_at"`
			Name       string `json:"name"`
			Conclusion string `json:"conclusion"`
		} `json:"workflow_runs"`
	}
	if err := fetch("https://api.github.com/repos/anonymous-writer/Veltrio/actions/runs?per_page=10", &runs); err == nil {
		for _, run := range runs.WorkflowRuns {
			if run.Conclusion == "success" {
				result.UpdatedAt = run.UpdatedAt
				result.RunURL = run.HTMLURL
				break
			}
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, result)
}
func (s *Server) downloads(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, s.manager.List())
	case http.MethodPost:
		var req struct {
			URL      string `json:"url"`
			Filename string `json:"filename"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Ungültige Anfrage", 400)
			return
		}
		item, err := s.manager.Add(req.URL, req.Filename)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, item)
	default:
		http.Error(w, "Methode nicht erlaubt", 405)
	}
}
func (s *Server) downloadAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/downloads/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodDelete && len(parts) == 1 {
		if err := s.manager.Delete(parts[0]); err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		w.WriteHeader(204)
		return
	}
	if r.Method == http.MethodPost && len(parts) == 2 {
		var err error
		switch parts[1] {
		case "pause":
			err = s.manager.Pause(parts[0])
		case "resume":
			err = s.manager.Resume(parts[0])
		default:
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	http.Error(w, "Methode nicht erlaubt", 405)
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}
