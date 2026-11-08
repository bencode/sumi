package syntax

import "testing"

func TestPrepare(t *testing.T) {
	tests := []struct {
		name, src, body, title string
	}{
		{"no front matter", "sequenceDiagram\nA->>B: hi", "sequenceDiagram\nA->>B: hi\n", ""},
		{"title", "---\ntitle: Hello\n---\nsequenceDiagram", "\n\n\nsequenceDiagram\n", "Hello"},
		{"quoted title", "---\ntitle: \"Hi: there\"\n---\nx", "\n\n\nx\n", "Hi: there"},
		{"nested config", "---\nconfig:\n  sequence:\n    mirrorActors: false\ntitle: T\n---\nx", "\n\n\n\n\n\nx\n", "T"},
		{"indented fences", "  ---\n  title: T\n  ---\nx", "\n\n\nx\n", "T"},
		{"an indented --- inside the front matter", "---\nconfig:\n  a: |\n    ---\ntitle: T\n---\nx", "\n\n\n\n\n\nx\n", "T"},
		{"unclosed", "---\ntitle: T\nx", "---\ntitle: T\nx\n", ""},
		{"CRLF", "---\r\ntitle: T\r\n---\r\nx\r\ny", "\n\n\nx\ny\n", "T"},
		{"comment", "a\n  %% note\nb: c %% d\ne", "a\n\nb: c %% d\ne\n", ""},
		{"a comment on the last line goes", "a\n%% b", "a\n\n", ""},
		{"directive", "%%{init: {\n'theme': 'dark'}}%%\nx", "\n\nx\n", ""},
		{"a %%{ with no word after it stays", "A %%{ -\nB", "A %%{ -\nB\n", ""},
		{"a directive without its close runs on", "%%{a: {1\nb", "\n\n", ""},
		{"what stands around a directive joins", "a %%{b c\nd\ne", "a d\n\ne\n", ""},
		{"a second word ends a directive", "%%{a b c}%%\nx", "c}%%\nx\n", ""},
		{"entity", "A->>B: I #9829; you #59;", "A->>B: I \uE0009829\uE001 you \uE00059\uE001\n", ""},
		{"html attribute", "A[<img src=\"x\" a=\"b\">] \"c\"", "A[<img src='x' a='b'>] \"c\"\n", ""},
		{"style with a color", "style A fill:#f00;\nB", "style A fill:#f00\nB\n", ""},
		{"classDef with a color", "classDef c fill:#123,color:#fff;", "classDef c fill:#123,color:#fff\n", ""},
	}
	for _, tt := range tests {
		body, title := Prepare(tt.src)
		if body != tt.body || title != tt.title {
			t.Errorf("%s: Prepare(%q) = %q, %q; want %q, %q", tt.name, tt.src, body, title, tt.body, tt.title)
		}
	}
}

func TestDecode(t *testing.T) {
	body, _ := Prepare("#9829; #quot; #nosuch; #0;")
	if got, want := Decode(body), "\u2665 \" &nosuch; \uFFFD\n"; got != want {
		t.Errorf("Decode = %q, want %q", got, want)
	}
}
