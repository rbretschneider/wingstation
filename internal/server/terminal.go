package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // same-origin enforced by browser; auth middleware handles access
	},
}

// resizeMessage is sent from xterm.js to resize the TTY.
type resizeMessage struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("id")
	if containerID == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Verify container is running
	inspect, err := s.execClient.ContainerInspect(ctx, containerID)
	if err != nil {
		slog.Error("terminal: inspect container", "id", containerID, "error", err)
		http.Error(w, "Container not found", http.StatusNotFound)
		return
	}
	if !inspect.State.Running {
		http.Error(w, "Container is not running", http.StatusBadRequest)
		return
	}

	// Create exec
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"/bin/sh"},
	}

	execResp, err := s.execClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		slog.Error("terminal: exec create", "id", containerID, "error", err)
		http.Error(w, "Failed to create exec", http.StatusInternalServerError)
		return
	}

	// Attach to exec
	attachResp, err := s.execClient.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		slog.Error("terminal: exec attach", "id", containerID, "error", err)
		http.Error(w, "Failed to attach to exec", http.StatusInternalServerError)
		return
	}

	// Upgrade to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("terminal: websocket upgrade", "error", err)
		attachResp.Close()
		return
	}

	slog.Info("terminal: session started", "container", containerID)

	var wg sync.WaitGroup
	execID := execResp.ID

	// exec stdout → WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := attachResp.Reader.Read(buf)
			if n > 0 {
				if writeErr := ws.WriteMessage(websocket.TextMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					slog.Debug("terminal: read from exec", "error", err)
				}
				return
			}
		}
	}()

	// WebSocket → exec stdin (text = stdin, binary = control)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msgType, msg, err := ws.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					slog.Debug("terminal: read from ws", "error", err)
				}
				return
			}

			switch msgType {
			case websocket.TextMessage:
				// stdin data
				if _, err := attachResp.Conn.Write(msg); err != nil {
					slog.Debug("terminal: write to exec", "error", err)
					return
				}
			case websocket.BinaryMessage:
				// control message (resize)
				var rm resizeMessage
				if err := json.Unmarshal(msg, &rm); err != nil {
					slog.Debug("terminal: parse control msg", "error", err)
					continue
				}
				if rm.Type == "resize" {
					if err := s.execClient.ContainerExecResize(context.Background(), execID, container.ResizeOptions{
						Height: rm.Rows,
						Width:  rm.Cols,
					}); err != nil {
						slog.Debug("terminal: resize", "error", err)
					}
				}
			}
		}
	}()

	wg.Wait()
	attachResp.Close()
	ws.Close()
	slog.Info("terminal: session ended", "container", containerID)
}
