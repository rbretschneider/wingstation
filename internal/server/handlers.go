package server

import (
	"log/slog"
	"net/http"

	"github.com/rbretschneider/wingstation/internal/docker"
	"github.com/rbretschneider/wingstation/internal/service"
)

// pageData is the common data passed to full page templates.
type pageData struct {
	Title  string
	Page   string
	Groups interface{}
	Host   interface{}
	Error  string
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	groups, err := s.containerService.ListGrouped(r.Context())
	if err != nil {
		slog.Error("listing containers", "error", err)
		s.renderError(w, "Failed to list containers", http.StatusInternalServerError)
		return
	}

	data := pageData{
		Title:  "Dashboard",
		Page:   "dashboard",
		Groups: groups,
	}

	s.renderPage(w, "dashboard", data)
}

func (s *Server) handleHost(w http.ResponseWriter, r *http.Request) {
	info, err := s.hostService.GetInfo(r.Context())

	data := pageData{
		Title: "Host Info",
		Page:  "host",
	}
	if err != nil {
		slog.Error("getting host info", "error", err)
		data.Error = "Failed to retrieve host information: " + err.Error()
	} else {
		data.Host = info
	}

	s.renderPage(w, "host", data)
}

func (s *Server) handlePartialContainers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := service.SearchParams{
		Query:  q.Get("q"),
		Status: q.Get("status"),
		Group:  q.Get("group"),
		Tag:    q.Get("tag"),
		Sort:   q.Get("sort"),
	}

	// If search or status/tag filters active, do a flat filtered search
	if params.Query != "" || params.Status != "" || params.Tag != "" {
		containers, err := s.containerService.Search(r.Context(), params)
		if err != nil {
			slog.Error("searching containers", "error", err)
			http.Error(w, "Search failed", http.StatusInternalServerError)
			return
		}
		// Wrap into a single pseudo-group for template compatibility
		groups := []docker.ContainerGroup{{
			Name:       "Search Results",
			Containers: containers,
		}}
		s.renderPartial(w, "container_list", pageData{Groups: groups})
		return
	}

	groups, err := s.containerService.ListGrouped(r.Context())
	if err != nil {
		slog.Error("listing containers", "error", err)
		http.Error(w, "Failed to list containers", http.StatusInternalServerError)
		return
	}

	// Apply group filter if specified
	if params.Group != "" {
		var filtered []docker.ContainerGroup
		for _, g := range groups {
			if g.Name == params.Group {
				filtered = append(filtered, g)
			}
		}
		s.renderPartial(w, "container_list", pageData{Groups: filtered})
		return
	}

	s.renderPartial(w, "container_list", pageData{Groups: groups})
}

func (s *Server) handlePartialContainerCard(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	detail, err := s.containerService.Inspect(r.Context(), id)
	if err != nil {
		slog.Error("inspecting container", "id", id, "error", err)
		http.Error(w, "Container not found", http.StatusNotFound)
		return
	}

	s.renderPartial(w, "container_card", detail)
}

func (s *Server) handlePartialDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	detail, err := s.containerService.Inspect(r.Context(), id)
	if err != nil {
		slog.Error("inspecting container", "id", id, "error", err)
		http.Error(w, "Container not found", http.StatusNotFound)
		return
	}

	s.renderPartial(w, "detail_panel", detail)
}

func (s *Server) renderPage(w http.ResponseWriter, page string, data pageData) {
	tmpl, ok := s.pages[page]
	if !ok {
		slog.Error("page template not found", "page", page)
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Execute the base layout, which calls {{template "content" .}} defined by the page
	if err := tmpl.ExecuteTemplate(w, "templates/layouts/base.html", data); err != nil {
		slog.Error("rendering page", "page", page, "error", err)
	}
}

func (s *Server) renderPartial(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.partials.ExecuteTemplate(w, name, data); err != nil {
		slog.Error("rendering partial", "name", name, "error", err)
		http.Error(w, "Render error", http.StatusInternalServerError)
	}
}

func (s *Server) renderError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	// Use dashboard page template as a fallback for error display
	if tmpl, ok := s.pages["dashboard"]; ok {
		data := pageData{
			Title: "Error",
			Page:  "dashboard",
			Error: msg,
		}
		if err := tmpl.ExecuteTemplate(w, "templates/layouts/base.html", data); err != nil {
			http.Error(w, msg, code)
		}
		return
	}
	http.Error(w, msg, code)
}
