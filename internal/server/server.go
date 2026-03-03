package server

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rbretschneider/wingstation/internal/config"
	"github.com/rbretschneider/wingstation/internal/docker"
	"github.com/rbretschneider/wingstation/internal/service"
	"github.com/rbretschneider/wingstation/web"
)

// Server is the WingStation HTTP server.
type Server struct {
	cfg              *config.Config
	containerService *service.ContainerService
	hostService      *service.HostService
	dockerClient     docker.ReadOnlyClient
	execClient       docker.ExecClient // nil when terminal is disabled
	pages            map[string]*template.Template // per-page template sets (base+partials+page)
	partials         *template.Template            // shared partials for htmx responses
	httpServer       *http.Server
	sse              *SSEBroker
}

// New creates a new Server with all dependencies wired.
// execClient may be nil when terminal feature is disabled.
func New(
	cfg *config.Config,
	containerService *service.ContainerService,
	hostService *service.HostService,
	dockerClient docker.ReadOnlyClient,
	execClient docker.ExecClient,
) (*Server, error) {
	pages, partials, err := parseTemplates(cfg.TerminalEnabled)
	if err != nil {
		return nil, fmt.Errorf("parsing templates: %w", err)
	}

	s := &Server{
		cfg:              cfg,
		containerService: containerService,
		hostService:      hostService,
		dockerClient:     dockerClient,
		execClient:       execClient,
		pages:            pages,
		partials:         partials,
		sse:              NewSSEBroker(),
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	var handler http.Handler = mux
	handler = s.recoveryMiddleware(handler)
	handler = s.loggingMiddleware(handler)
	if cfg.AuthEnabled {
		handler = s.basicAuthMiddleware(handler)
	}

	s.httpServer = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Static files
	staticFS, _ := fs.Sub(web.Files, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Pages
	mux.HandleFunc("GET /{$}", s.handleDashboard)
	mux.HandleFunc("GET /host", s.handleHost)

	// Partials (htmx)
	mux.HandleFunc("GET /partials/containers", s.handlePartialContainers)
	mux.HandleFunc("GET /partials/container-card", s.handlePartialContainerCard)
	mux.HandleFunc("GET /partials/detail", s.handlePartialDetail)

	// API (JSON)
	mux.HandleFunc("GET /api/containers", s.handleAPIContainers)
	mux.HandleFunc("GET /api/containers/{id}", s.handleAPIContainerDetail)
	mux.HandleFunc("GET /api/host", s.handleAPIHost)

	// SSE
	mux.HandleFunc("GET /events", s.handleSSE)

	// Terminal WebSocket (only when exec client is available)
	if s.execClient != nil {
		mux.HandleFunc("GET /ws/terminal", s.handleTerminalWS)
	}

	// Health
	mux.HandleFunc("GET /healthz", s.handleHealthz)
}

// Start begins listening and serving. It blocks until the server is shut down.
func (s *Server) Start() error {
	// Start SSE event watcher
	if s.cfg.SSEEnabled {
		go s.watchDockerEvents()
	}
	s.sse.Start()

	slog.Info("WingStation listening", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.sse.Stop()
	return s.httpServer.Shutdown(ctx)
}

func buildFuncMap(terminalEnabled bool) template.FuncMap {
	return template.FuncMap{
		"formatUptime":    formatUptime,
		"formatBytes":     formatBytes,
		"formatNanoCPU":   formatNanoCPU,
		"maskValue":       maskValue,
		"isSensitiveKey":  isSensitiveKey,
		"isSensitivePath": isSensitivePath,
		"statusColor":     statusColor,
		"joinStrings":     joinStrings,
		"percentage":      percentage,
		"firstN":          firstN,
		"sub":             func(a, b int) int { return a - b },
		"terminalEnabled": func() bool { return terminalEnabled },
	}
}

// parseTemplates builds per-page template sets and a shared partials-only template.
// Each page set = base layout + all partials + that specific page template.
// This avoids the "last define wins" problem with multiple pages defining {{define "content"}}.
func parseTemplates(terminalEnabled bool) (map[string]*template.Template, *template.Template, error) {
	fm := buildFuncMap(terminalEnabled)

	// Read shared files: base layout + all partials
	sharedFiles := []string{"templates/layouts/base.html"}
	partialMatches, err := fs.Glob(web.Files, "templates/partials/*.html")
	if err != nil {
		return nil, nil, err
	}
	sharedFiles = append(sharedFiles, partialMatches...)

	sharedContents := make(map[string]string, len(sharedFiles))
	for _, f := range sharedFiles {
		data, err := fs.ReadFile(web.Files, f)
		if err != nil {
			return nil, nil, fmt.Errorf("reading %s: %w", f, err)
		}
		sharedContents[f] = string(data)
	}

	// Build a partials-only template set for htmx partial rendering
	partials := template.New("").Funcs(fm)
	for _, f := range partialMatches {
		if _, err := partials.New(f).Parse(sharedContents[f]); err != nil {
			return nil, nil, fmt.Errorf("parsing partial %s: %w", f, err)
		}
	}

	// Find all page templates
	pageMatches, err := fs.Glob(web.Files, "templates/pages/*.html")
	if err != nil {
		return nil, nil, err
	}

	pages := make(map[string]*template.Template, len(pageMatches))
	for _, pagePath := range pageMatches {
		// Extract page name: "templates/pages/dashboard.html" -> "dashboard"
		name := strings.TrimPrefix(pagePath, "templates/pages/")
		name = strings.TrimSuffix(name, ".html")

		pageData, err := fs.ReadFile(web.Files, pagePath)
		if err != nil {
			return nil, nil, fmt.Errorf("reading page %s: %w", pagePath, err)
		}

		// Build a fresh template set: shared files + this page
		t := template.New("base").Funcs(fm)
		for f, content := range sharedContents {
			if _, err := t.New(f).Parse(content); err != nil {
				return nil, nil, fmt.Errorf("parsing %s for page %s: %w", f, name, err)
			}
		}
		if _, err := t.New(pagePath).Parse(string(pageData)); err != nil {
			return nil, nil, fmt.Errorf("parsing page %s: %w", pagePath, err)
		}

		pages[name] = t
	}

	return pages, partials, nil
}

// Template helper functions

func formatUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case b >= tb:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(tb))
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatNanoCPU(n int64) string {
	return fmt.Sprintf("%.2f", float64(n)/1e9)
}

func maskValue(key, value string) string {
	if isSensitiveKey(key) {
		if len(value) <= 4 {
			return "****"
		}
		return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
	}
	return value
}

var sensitiveKeys = []string{
	"password", "passwd", "secret", "token", "key", "apikey",
	"api_key", "auth", "credential", "private",
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

var sensitivePaths = []string{
	"/var/run/docker.sock",
	"/etc/shadow",
	"/etc/passwd",
	"/root",
	"/proc",
	"/sys",
}

func isSensitivePath(path string) bool {
	for _, sp := range sensitivePaths {
		if strings.HasPrefix(path, sp) {
			return true
		}
	}
	return false
}

func statusColor(state string) string {
	switch state {
	case "running":
		return "bg-green-500"
	case "paused":
		return "bg-yellow-500"
	case "restarting":
		return "bg-blue-500"
	case "exited", "dead":
		return "bg-red-500"
	case "created":
		return "bg-gray-400"
	default:
		return "bg-gray-400"
	}
}

func joinStrings(strs []string, sep string) string {
	return strings.Join(strs, sep)
}

func percentage(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}

// firstN returns the first n items from a PortMapping slice.
func firstN(ports []docker.PortMapping, n int) []docker.PortMapping {
	if len(ports) <= n {
		return ports
	}
	return ports[:n]
}
