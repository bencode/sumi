package main

import (
	"strings"
	"testing"
)

func TestCheckReportsFileLines(t *testing.T) {
	doc := strings.Join([]string{
		"# Doc",                 // 1
		"",                      // 2
		"```mermaid",            // 3
		"flowchart TD",          // 4
		"  A[路径/a(b): c] --> B", // 5: "(" is rune 9
		"```",                   // 6
		"",                      // 7
		"- item",                // 8
		"  ```mermaid",          // 9: fence indented 2
		"  flowchart LR",        // 10
		"    A --> B --> C",     // 11
		"    S1 -> S2",          // 12: "-" is rune 6 of the block, 8 of the file
		"  ```",                 // 13
		"",                      // 14
		"~~~mermaid",            // 15
		"---",                   // 16
		"title: t",              // 17
		"---",                   // 18
		"stateDiagram-v2",       // 19
		"  S1 -> S2",            // 20
		"~~~",                   // 21
		"```js",                 // 22
		"not mermaid",           // 23
		"```",                   // 24
		"````mermaid",           // 25
		"sequenceDiagram",       // 26
		"  Alice->>Bob: hi",     // 27
		"```",                   // 28: shorter than the fence, so content
		"````",                  // 29
		"```mermaid",            // 30
		"flowchart LR",          // 31
		"  A[x] --> B(y",        // 32: never closed, so its end is the file's
	}, "\n")
	rs := check("doc.md", doc)
	want := []string{"doc.md:5:9:", "doc.md:12:8:", "doc.md:20:6:", "doc.md:28:4:", "doc.md:32:1:"}
	if len(rs) != len(want) {
		t.Fatalf("got %d diagrams, want %d", len(rs), len(want))
	}
	for i, r := range rs {
		if r.err == nil {
			t.Errorf("diagram %d: no error, want one at %s", i+1, want[i])
			continue
		}
		if got := r.report("doc.md"); !strings.HasPrefix(got, want[i]) {
			t.Errorf("diagram %d: got %q, want it at %s", i+1, got, want[i])
		}
	}
}

func TestCheckReadsOtherFilesWhole(t *testing.T) {
	rs := check("a.mmd", "flowchart LR\r\n  A --> B\r\n")
	if len(rs) != 1 || rs[0].err != nil || rs[0].first != 1 {
		t.Fatalf("got %+v, want one good diagram from line 1", rs)
	}
	if rs = check("a.mmd", "flowchart LR\n  A --> B(\n"); len(rs) != 1 || !strings.HasPrefix(rs[0].report("a.mmd"), "a.mmd:3:1:") {
		t.Fatalf("got %+v, want an error at the end of the file", rs)
	}
}

func TestCaretUnderColumn(t *testing.T) {
	got := check("a.mmd", "flowchart TD\n\tA[路径/a(b)]")[0].report("a.mmd")
	// A tab stays a tab; "A[" takes two columns, "路径" four, "/a" two.
	if want := "\n    \t        ^\n"; !strings.HasSuffix(got, want) {
		t.Errorf("got %q, want it to end %q", got, want)
	}
}
