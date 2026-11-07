package flowchart

import "strings"

// endLink is what the end of a link says (destructEndLink).
func endLink(s string) (kind, stroke string) {
	s = trim(s)
	line := s[:len(s)-1]
	kind = "arrow_open"
	for _, end := range []struct{ last, first, name string }{
		{"x", "x", "arrow_cross"},
		{">", "<", "arrow_point"},
		{"o", "o", "arrow_circle"},
	} {
		if !strings.HasSuffix(s, end.last) {
			continue
		}
		kind = end.name
		if strings.HasPrefix(s, end.first) {
			kind = "double_" + kind
			line = line[1:]
		}
	}
	stroke = "normal"
	switch {
	case strings.Contains(line, "."):
		stroke = "dotted"
	case strings.HasPrefix(line, "="):
		stroke = "thick"
	case strings.HasPrefix(line, "~"):
		stroke = "invisible"
	}
	return kind, stroke
}

// startLink is what the start of a link with text says (destructStartLink).
func startLink(s string) (kind, stroke string) {
	s = trim(s)
	kind = "arrow_open"
	switch s[0] {
	case '<':
		kind = "arrow_point"
	case 'x':
		kind = "arrow_cross"
	case 'o':
		kind = "arrow_circle"
	}
	stroke = "normal"
	if strings.Contains(s, "=") {
		stroke = "thick"
	}
	if strings.Contains(s, ".") {
		stroke = "dotted"
	}
	return kind, stroke
}

// invalid is the type and stroke of a link whose two ends disagree. Mermaid draws it.
const invalid = "INVALID"

// destructLink is the type and stroke of a link from its end, and from its start if it
// has text in it.
func destructLink(end, start string) (kind, stroke string) {
	kind, stroke = endLink(end)
	if start == "" {
		return kind, stroke
	}
	from, fromStroke := startLink(start)
	if fromStroke != stroke {
		return invalid, invalid
	}
	if from == "arrow_open" {
		return kind, stroke
	}
	if from != kind {
		return invalid, invalid
	}
	return "double_" + from, stroke
}

// markers are the markers at the two ends of a link (destructEdgeType, getData).
func markers(kind string) (start, end string) {
	if kind == "arrow_open" {
		return "none", "none"
	}
	if name, ok := strings.CutPrefix(kind, "double_"); ok {
		return name, name
	}
	if kind == invalid {
		return "none", "arrow_point"
	}
	return "none", kind
}
