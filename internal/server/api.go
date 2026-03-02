package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/rbretschneider/wingstation/internal/service"
)

func (s *Server) handleAPIContainers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := service.SearchParams{
		Query:  q.Get("q"),
		Status: q.Get("status"),
		Group:  q.Get("group"),
		Tag:    q.Get("tag"),
		Sort:   q.Get("sort"),
	}

	containers, err := s.containerService.Search(r.Context(), params)
	if err != nil {
		slog.Error("API: listing containers", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list containers"})
		return
	}

	writeJSON(w, http.StatusOK, containers)
}

func (s *Server) handleAPIContainerDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing container id"})
		return
	}

	detail, err := s.containerService.Inspect(r.Context(), id)
	if err != nil {
		slog.Error("API: inspecting container", "id", id, "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "container not found"})
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleAPIHost(w http.ResponseWriter, r *http.Request) {
	info, err := s.hostService.GetInfo(r.Context())
	if err != nil {
		slog.Error("API: getting host info", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get host info"})
		return
	}

	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	_, err := s.dockerClient.Ping(r.Context())
	if err != nil {
		slog.Error("healthz: Docker ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writing JSON response", "error", err)
	}
}
