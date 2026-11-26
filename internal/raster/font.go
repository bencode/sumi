package raster

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/image/font/sfnt"

	"github.com/bencode/sumi/scene"
)

// Fonts are the system's typefaces a scene's text is drawn in: for each
// style, a list to take each character from the first that has it.
type Fonts struct {
	styles [4][]*sfnt.Font // by style: regular, bold, italic, bold italic
	mono   []*sfnt.Font
}

// The files looked for, the first found of each list taken. Arial draws what
// scene.Arial measures; the fallbacks draw the characters Arial lacks, as CJK.
var (
	styleFiles = [4][]string{
		{"/System/Library/Fonts/Supplemental/Arial.ttf", "/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf", "/usr/share/fonts/liberation-sans/LiberationSans-Regular.ttf"},
		{"/System/Library/Fonts/Supplemental/Arial Bold.ttf", "/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf", "/usr/share/fonts/liberation-sans/LiberationSans-Bold.ttf"},
		{"/System/Library/Fonts/Supplemental/Arial Italic.ttf", "/usr/share/fonts/truetype/liberation/LiberationSans-Italic.ttf", "/usr/share/fonts/liberation-sans/LiberationSans-Italic.ttf"},
		{"/System/Library/Fonts/Supplemental/Arial Bold Italic.ttf", "/usr/share/fonts/truetype/liberation/LiberationSans-BoldItalic.ttf", "/usr/share/fonts/liberation-sans/LiberationSans-BoldItalic.ttf"},
	}
	monoFiles     = []string{"/System/Library/Fonts/Menlo.ttc", "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf", "/usr/share/fonts/dejavu-sans-mono-fonts/DejaVuSansMono.ttf"}
	fallbackFiles = []string{
		"/System/Library/Fonts/Hiragino Sans GB.ttc",
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
	}
)

// LoadFonts loads the fonts found on the system. It fails when no regular
// font is found, saying where it looked.
func LoadFonts() (*Fonts, error) {
	var fonts Fonts
	var fallbacks []*sfnt.Font
	for _, path := range fallbackFiles {
		f, err := load(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		fallbacks = append(fallbacks, f)
	}
	for i, paths := range styleFiles {
		f, err := first(paths)
		if err != nil {
			return nil, err
		}
		if f == nil && i == 0 {
			return nil, fmt.Errorf("no font to draw text with; looked for %s", strings.Join(paths, ", "))
		}
		if f == nil {
			fonts.styles[i] = fonts.styles[0]
			continue
		}
		fonts.styles[i] = append([]*sfnt.Font{f}, fallbacks...)
	}
	mono, err := first(monoFiles)
	if err != nil {
		return nil, err
	}
	fonts.mono = fonts.styles[0]
	if mono != nil {
		fonts.mono = append([]*sfnt.Font{mono}, fallbacks...)
	}
	return &fonts, nil
}

// chain is the list of fonts to draw text in f with.
func (fs *Fonts) chain(f scene.Font) []*sfnt.Font {
	if f.Mono {
		return fs.mono
	}
	i := 0
	if f.Bold {
		i |= 1
	}
	if f.Italic {
		i |= 2
	}
	return fs.styles[i]
}

// first is the font of the first of paths that exists, or nil if none does.
func first(paths []string) (*sfnt.Font, error) {
	for _, path := range paths {
		f, err := load(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		return f, err
	}
	return nil, nil
}

// load reads the font file at path; of a collection, its first font.
func load(path string) (*sfnt.Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(data, []byte("ttcf")) {
		f, err := sfnt.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return f, nil
	}
	c, err := sfnt.ParseCollection(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	f, err := c.Font(0)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}
