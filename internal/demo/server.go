// Package demo serves the interactive examples' rendering API.
package demo

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
	"github.com/bencode/sumi/svg"
)

const maxRequestBytes = 64 << 10

type renderRequest struct {
	Source string `json:"source"`
}

type renderResponse struct {
	LightSVG string  `json:"lightSVG"`
	DarkSVG  string  `json:"darkSVG"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
}

type renderError struct {
	Message     string `json:"message"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Unsupported bool   `json:"unsupported,omitempty"`
}

// NewHandler registers the rendering API and a process health check.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/render", handleRender)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, renderError{Message: "Endpoint not found."})
	})
	return mux
}

func handleRender(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	defer func() {
		if failure := recover(); failure != nil {
			slog.Error("diagram rendering panicked", "panic", failure, "stack", string(debug.Stack()))
			writeJSON(w, http.StatusInternalServerError, renderError{Message: "The diagram could not be rendered."})
		}
	}()
	source, err := readSource(w, r)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeJSON(w, http.StatusRequestEntityTooLarge, renderError{Message: "The request exceeds 64 KiB. Shorten the source and try again."})
			return
		}
		writeJSON(w, http.StatusBadRequest, renderError{Message: "Send one JSON object with a string source field."})
		return
	}
	if strings.TrimSpace(source) == "" {
		writeJSON(w, http.StatusBadRequest, renderError{Message: "Enter a Mermaid diagram to start rendering."})
		return
	}
	writeDiagram(w, source)
}

func writeDiagram(w http.ResponseWriter, source string) {
	s, err := mermaid.Layout(source, mermaid.Options{})
	if err != nil {
		writeLayoutError(w, err)
		return
	}
	if s.Width <= 0 || s.Height <= 0 {
		writeJSON(w, http.StatusUnprocessableEntity, renderError{Message: "The diagram has no drawable content."})
		return
	}
	if math.IsNaN(s.Width) || math.IsNaN(s.Height) || math.IsInf(s.Width, 0) || math.IsInf(s.Height, 0) {
		slog.Error("diagram has invalid dimensions", "width", s.Width, "height", s.Height)
		writeJSON(w, http.StatusInternalServerError, renderError{Message: "The diagram could not be rendered."})
		return
	}
	writeJSON(w, http.StatusOK, renderResponse{
		LightSVG: string(svg.Render(s, scene.Light)), DarkSVG: string(svg.Render(s, scene.Dark)),
		Width: s.Width, Height: s.Height,
	})
}

func readSource(w http.ResponseWriter, r *http.Request) (string, error) {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	var input renderRequest
	if err := decoder.Decode(&input); err != nil {
		return "", err
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return "", err
		}
		return "", errors.New("multiple JSON values")
	}
	return input.Source, nil
}

func writeLayoutError(w http.ResponseWriter, err error) {
	var sourceError *mermaid.Error
	if errors.As(err, &sourceError) {
		status := http.StatusBadRequest
		if sourceError.IsUnsupported() {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, renderError{Message: sourceError.Msg, Line: sourceError.Line,
			Column: sourceError.Col, Unsupported: sourceError.IsUnsupported()})
		return
	}
	slog.Error("diagram rendering failed", "error", err)
	writeJSON(w, http.StatusInternalServerError, renderError{Message: "The diagram could not be rendered."})
}

func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	writeJSON(w, http.StatusMethodNotAllowed, renderError{Message: "Use " + method + " for this endpoint."})
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	body, err := json.Marshal(value)
	if err != nil {
		slog.Error("could not encode response", "error", err)
		status = http.StatusInternalServerError
		body = []byte(`{"message":"The response could not be encoded."}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		slog.Error("could not write response", "error", err)
	}
}
