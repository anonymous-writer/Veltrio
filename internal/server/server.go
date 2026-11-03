package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"veltrio/internal/downloader"
)

//go:embed web/*
var embeddedWeb embed.FS

type Server struct {
	manager *downloader.Manager
	assets  fs.FS
	mux     *http.ServeMux
}

func New(stateFile, downloadDir string) (*Server, error) {
	assets, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		return nil, err
	}
	m, err := downloader.New(stateFile, downloadDir)
	if err != nil {
		return nil, err
	}
	s := &Server{manager: m, assets: assets, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}
func (s *Server) Handler() http.Handler { return s.mux }
func (s *Server) routes() {
	s.mux.HandleFunc("/api/downloads", s.downloads)
	s.mux.HandleFunc("/api/downloads/", s.downloadAction)
	s.mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "name": "Veltrio"})
	})
	s.mux.HandleFunc("/", s.frontend)
}
func (s *Server) frontend(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}
	if _, err := s.assets.Open(name); err != nil {
		name = "index.html"
	}
	http.FileServer(http.FS(s.assets)).ServeHTTP(w, r)
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
