package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Server struct {
	config     Config
	players    *PlayerSlots
	controller DeviceController
	httpServer *http.Server
}

func NewServer(config Config, controller DeviceController) *Server {
	server := &Server{
		config:     config,
		players:    NewPlayerSlots(config.MaxPlayers),
		controller: controller,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.health)
	mux.HandleFunc("/ws", server.websocket)
	mux.HandleFunc("/", server.static)

	server.httpServer = &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}

	return server
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Close() error {
	if s.controller != nil {
		return s.controller.Close()
	}
	return nil
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	type healthResponse struct {
		OK bool `json:"ok"`
	}

	writeJSON(w, http.StatusOK, healthResponse{OK: true})
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(s.config.PublicDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "Frontend has not been built into public dir", http.StatusNotFound)
		return
	}

	if requestedPath := strings.TrimPrefix(r.URL.Path, "/"); requestedPath != "" {
		if stat, err := os.Stat(filepath.Join(s.config.PublicDir, requestedPath)); err == nil && !stat.IsDir() {
			http.FileServer(http.Dir(s.config.PublicDir)).ServeHTTP(w, r)
			return
		}
	}

	http.ServeFile(w, r, indexPath)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Warn("failed to write json response", "error", err)
	}
}

// PlayerSlots tracks WebSocket player assignments for the server. Device
// controllers receive the assigned player number but do not share this state.
type PlayerSlots struct {
	mu         sync.Mutex
	maxPlayers int
	occupied   map[int]bool
}

func NewPlayerSlots(maxPlayers int) *PlayerSlots {
	return &PlayerSlots{
		maxPlayers: maxPlayers,
		occupied:   make(map[int]bool),
	}
}

func (p *PlayerSlots) Assign() (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for player := 1; player <= p.maxPlayers; player++ {
		if !p.occupied[player] {
			p.occupied[player] = true
			return player, true
		}
	}
	return 0, false
}

func (p *PlayerSlots) Release(player int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.occupied, player)
}
