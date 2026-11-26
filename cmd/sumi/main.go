// Command sumi checks Mermaid diagrams and renders them as images, the way
// sumi lays them out: the diagrams of a Markdown file's mermaid code blocks,
// or of a file of Mermaid source.
//
//	sumi check FILE...
//	sumi render FILE... -o DIR [-format png|svg] [-scale N]
//
// Errors are reported as FILE:LINE:COL: message, in the lines of FILE. The exit
// status is 0 when every diagram is good, 1 when one has an error, and 2 when the
// command cannot run: bad usage, a file it cannot read or write, or no font.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bencode/sumi/internal/raster"
	"github.com/bencode/sumi/scene"
	"github.com/bencode/sumi/svg"
)

const usage = `usage:
  sumi check FILE...
  sumi render FILE... -o DIR [-format png|svg] [-scale N]

FILE is a Markdown file (.md), whose mermaid code blocks are checked, or a
file of Mermaid source (any other name). render checks first and writes
nothing unless every diagram is good, then writes DIR/<name>-01.png, ... and
prints a line for each: its number, where its source starts, its image, and
its size in CSS pixels.
`

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	fs := flag.NewFlagSet("sumi "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	var out, format string
	var scale float64
	if args[0] == "render" {
		fs.StringVar(&out, "o", "", "the directory to write images to")
		fs.StringVar(&format, "format", "png", "the image format: png or svg")
		fs.Float64Var(&scale, "scale", 2, "pixels per CSS pixel of a PNG")
	} else if args[0] != "check" {
		fmt.Fprint(stderr, usage)
		return 2
	}
	files, err := parse(fs, args[1:])
	if err != nil {
		return 2
	}
	if len(files) == 0 || (args[0] == "render" && (out == "" || (format != "png" && format != "svg") || scale <= 0)) {
		fmt.Fprint(stderr, usage)
		return 2
	}
	checked := make([][]result, len(files))
	failed := false
	for i, name := range files {
		text, err := os.ReadFile(name)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		checked[i] = check(name, string(text))
		for _, r := range checked[i] {
			if r.err != nil {
				fmt.Fprint(stderr, r.report(name))
				failed = true
			}
		}
	}
	if failed {
		return 1
	}
	if args[0] == "render" {
		if err := render(files, checked, out, format, scale, stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	return 0
}

// parse parses args with fs, flags before, between or after the files, and
// returns the files.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var files []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return files, nil
		}
		files = append(files, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// render writes an image of each diagram of files into dir, numbered within its
// file, and prints a line for each.
func render(files []string, checked [][]result, dir, format string, scale float64, stdout io.Writer) error {
	var fonts *raster.Fonts
	if format == "png" {
		var err error
		if fonts, err = raster.LoadFonts(); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	written := map[string]string{}
	for i, name := range files {
		stem := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		width := max(2, len(fmt.Sprint(len(checked[i]))))
		for n, r := range checked[i] {
			num := fmt.Sprintf("%0*d", width, n+1)
			path := filepath.Join(dir, stem+"-"+num+"."+format)
			if other, ok := written[path]; ok {
				return fmt.Errorf("%s and %s both write %s; render them into different directories", other, name, path)
			}
			written[path] = name
			if err := write(path, r.scene, format, scale, fonts); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "%s  %s:%d  %s  %.0fx%.0f\n", num, name, r.first, path, r.scene.Width, r.scene.Height)
		}
	}
	return nil
}

func write(path string, s *scene.Scene, format string, scale float64, fonts *raster.Fonts) error {
	if format == "svg" {
		return os.WriteFile(path, svg.Render(s, scene.Light), 0o644)
	}
	img, err := raster.Render(s, scene.Light, scale, fonts)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
