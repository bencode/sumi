package demo_test

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bencode/sumi/internal/demo"
)

func TestExamples(t *testing.T) {
	sources := os.DirFS("../../docs/examples")
	data, err := fs.ReadFile(sources, "catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Groups []struct {
			ID       string
			Examples []struct{ ID string }
		}
	}
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Groups) == 0 {
		t.Fatal("the example catalog has no categories")
	}
	groups := map[string]bool{}
	for _, group := range catalog.Groups {
		if group.ID == "" || groups[group.ID] {
			t.Fatalf("empty or duplicate category ID %q", group.ID)
		}
		groups[group.ID] = true
		if len(group.Examples) == 0 {
			t.Fatalf("category %q has no examples", group.ID)
		}
		examples := map[string]bool{}
		for _, example := range group.Examples {
			if example.ID == "" || examples[example.ID] {
				t.Fatalf("empty or duplicate example ID %q in %q", example.ID, group.ID)
			}
			examples[example.ID] = true
			t.Run(group.ID+"/"+example.ID, func(t *testing.T) {
				t.Parallel()
				src, err := fs.ReadFile(sources, group.ID+"/"+example.ID+".mmd")
				if err != nil {
					t.Fatal(err)
				}
				checkRender(t, string(src))
			})
		}
	}
}

func checkRender(t *testing.T, source string) {
	t.Helper()
	input, err := json.Marshal(map[string]string{"source": source})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "/api/render", bytes.NewReader(input))
	w := httptest.NewRecorder()
	demo.NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("render status %d: %s", w.Code, w.Body)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"lightSVG", "darkSVG", "width", "height"} {
		if fields[key] == nil {
			t.Fatalf("missing response field %q", key)
		}
	}
	var result struct {
		LightSVG, DarkSVG string
		Width, Height     float64
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Width <= 0 || result.Height <= 0 || result.LightSVG == result.DarkSVG {
		t.Fatalf("invalid dimensions or identical themes: %g × %g", result.Width, result.Height)
	}
	for _, svg := range []string{result.LightSVG, result.DarkSVG} {
		var doc struct {
			XMLName xml.Name `xml:"svg"`
			Width   float64  `xml:"width,attr"`
			Height  float64  `xml:"height,attr"`
		}
		if err := xml.Unmarshal([]byte(svg), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.Width <= 0 || doc.Height <= 0 {
			t.Fatalf("invalid SVG size: %g × %g", doc.Width, doc.Height)
		}
	}
}

func TestRenderErrors(t *testing.T) {
	for _, tt := range []struct {
		name, body  string
		status      int
		line, col   int
		unsupported bool
	}{
		{"invalid JSON", "{", 400, 0, 0, false},
		{"wrong source type", `{"source":123}`, 400, 0, 0, false},
		{"multiple JSON values", `{} {}`, 400, 0, 0, false},
		{"empty source", `{"source":""}`, 400, 0, 0, false},
		{"source error after front matter", `{"source":"---\ntitle: x\n---\nsequenceDiagram\nloop\nend\nend"}`, 400, 7, 1, false},
		{"unsupported diagram", `{"source":"pie\n  a: 1"}`, 422, 1, 1, true},
		{"empty scene", `{"source":"sequenceDiagram"}`, 422, 0, 0, false},
		{"large body", `{"source":"` + strings.Repeat("a", 64<<10) + `"}`, 413, 0, 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			demo.NewHandler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/render", strings.NewReader(tt.body)))
			if w.Code != tt.status {
				t.Fatalf("status %d, want %d: %s", w.Code, tt.status, w.Body)
			}
			var failure struct {
				Message      string
				Line, Column int
				Unsupported  bool
			}
			if err := json.Unmarshal(w.Body.Bytes(), &failure); err != nil {
				t.Fatal(err)
			}
			if failure.Message == "" || failure.Line != tt.line || failure.Column != tt.col || failure.Unsupported != tt.unsupported {
				t.Fatalf("unexpected error details: %+v", failure)
			}
		})
	}
}

func TestRouting(t *testing.T) {
	handler := demo.NewHandler()
	for _, tt := range []struct {
		method, path, allow string
		status              int
	}{
		{http.MethodGet, "/healthz", "", 200},
		{http.MethodPost, "/healthz", http.MethodGet, 405},
		{http.MethodGet, "/api/render", http.MethodPost, 405},
		{http.MethodGet, "/api/missing", "", 404},
	} {
		t.Run(tt.method+tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			if w.Code != tt.status || w.Header().Get("Allow") != tt.allow {
				t.Fatalf("status %d, allow %q: %s", w.Code, w.Header().Get("Allow"), w.Body)
			}
			if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("unexpected content type: %s", w.Header().Get("Content-Type"))
			}
			if tt.path == "/healthz" && tt.status == 200 {
				var health map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
					t.Fatal(err)
				}
				if health["status"] != "ok" {
					t.Fatalf("unexpected health response: %s", w.Body)
				}
			}
		})
	}
}
