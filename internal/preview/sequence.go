package main

import (
	"errors"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
	"github.com/bencode/sumi/svg"
)

// sequencePage lays out every .mmd file under dir, writes each as SVG in
// the light and dark palettes to out/sequence, and writes out/sequence.html:
// a row for each diagram, with its source to compare against Mermaid and
// its images, or the error that failed it; then the diagrams refused and
// why.
func sequencePage(dir, out string) error {
	if err := os.MkdirAll(filepath.Join(out, "sequence"), 0o755); err != nil {
		return err
	}
	var rows, refused strings.Builder
	n, skipped, failed := 0, 0, 0
	err := filepath.WalkDir(dir, func(path string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() || !strings.HasSuffix(path, ".mmd") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		s, err := mermaid.Layout(string(src), mermaid.Options{})
		var e *mermaid.Error
		if errors.As(err, &e) && e.IsUnsupported() {
			skipped++
			fmt.Fprintf(&refused, "<li>%s: %s</li>\n", html.EscapeString(rel), html.EscapeString(e.Error()))
			return nil
		}
		fmt.Fprintf(&rows, "<tr><td><b>%s</b><pre>%s</pre></td>", html.EscapeString(rel), html.EscapeString(string(src)))
		if err != nil {
			failed++
			fmt.Fprintf(&rows, `<td colspan=2 style="color:#c00">%s</td></tr>`+"\n", html.EscapeString(err.Error()))
			return nil
		}
		n++
		name := strings.ReplaceAll(strings.TrimSuffix(rel, ".mmd"), string(filepath.Separator), "-")
		for _, th := range themes {
			file := "sequence/" + name + "-" + th.name + ".svg"
			if err := os.WriteFile(filepath.Join(out, file), svg.Render(s, th.p), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(&rows, `<td style="background:%s"><img src="%s"></td>`, th.bg, file)
		}
		rows.WriteString("</tr>\n")
		return nil
	})
	if err != nil {
		return err
	}
	page := "<!doctype html><meta charset=utf-8><title>sumi: sequence diagrams</title>\n" +
		"<style>body{margin:16px;font:14px sans-serif}td{padding:16px;vertical-align:top}" +
		"pre{font-size:12px;max-width:360px;white-space:pre-wrap}</style>\n" +
		fmt.Sprintf("<p>Mermaid's sequence test diagrams: %d laid out, %d failed, %d not supported yet.</p>\n", n, failed, skipped) +
		"<table>\n" + rows.String() + "</table>\n<h2>Not supported yet</h2>\n<ul>\n" + refused.String() + "</ul>\n"
	return os.WriteFile(filepath.Join(out, "sequence.html"), []byte(page), 0o644)
}

var themes = []struct {
	name string
	p    scene.Palette
	bg   string
}{{"light", scene.Light, "#ffffff"}, {"dark", scene.Dark, "#0d1117"}}
