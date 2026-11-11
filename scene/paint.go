package scene

import "math"

// A Color is an sRGB color with alpha.
type Color struct{ R, G, B, A uint8 }

// A Token names a color of the theme by what it is for, not by what it looks
// like, so a dark theme gives it a color of its own.
type Token uint8

const (
	None Token = iota // no token: the paint's own Color, or nothing
	TextColor
	LineColor // edges, lifelines
	NodeFill
	NodeStroke
	GroupFill
	GroupStroke
	NoteFill
	NoteStroke
	Background // the canvas the diagram is drawn on
	Highlight  // a background that sets part of a diagram apart: a sequence diagram's rect

	// A gantt diagram's. The sections' rows and the ticks are see-through, over the grid
	// and the excluded days.
	SectionFill      // the rows of the first section and every fourth after
	SectionFill2     // the rows of the third section and every fourth after
	SectionAltFill   // the rows of the sections between
	TaskFill         // a task's bar
	TaskStroke       // the outline of a task's bar
	ActiveTaskFill   // an active task's bar, crit or not
	ActiveTaskStroke // the outline of an active task's bar
	DoneTaskFill     // a done task's bar, crit or not
	DoneTaskStroke   // the outline of a done task's bar
	CritTaskFill     // a crit task's bar
	CritTaskStroke   // the outline of a crit task's bar, active, done or neither
	TaskText         // the text in a task's bar or a crit task's
	StatusTaskText   // the text in an active or a done task's bar, and beside an active one's
	LinkText         // the text of a task a click statement names
	VertLine         // a vert's line and its text
	TodayLine        // the line at the time now
	ExcludeFill      // the days a diagram excludes
	Tick             // the grid's lines and their labels
)

// A Paint is how something is filled or stroked: in the theme's color for
// Token, or, when Token is None, in Color, as a diagram's author set it. The
// zero Paint paints nothing.
type Paint struct {
	Token Token
	Color Color
}

// Use paints in the theme's color for t.
func Use(t Token) Paint { return Paint{Token: t} }

// Fixed paints in c, whatever the theme.
func Fixed(c Color) Paint { return Paint{Color: c} }

// A Palette is a theme's colors, by token.
type Palette map[Token]Color

// Resolve is the color p paints in, or false when it paints nothing: when it
// has no token and its color is transparent, or its token is missing from pal.
func (pal Palette) Resolve(p Paint) (Color, bool) {
	if p.Token == None {
		return p.Color, p.Color.A != 0
	}
	c, ok := pal[p.Token]
	return c, ok
}

func rgb(hex uint32) Color {
	return Color{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255}
}

// rgba is the color hex, opacity of 255 opaque.
func rgba(hex uint32, opacity uint8) Color {
	c := rgb(hex)
	c.A = opacity
	return c
}

// Light is sumi's own theme on a light background.
var Light = Palette{
	TextColor:   rgb(0x1f2328),
	LineColor:   rgb(0x57606a),
	NodeFill:    rgb(0xf6f8fa),
	NodeStroke:  rgb(0x8c959f),
	GroupFill:   rgb(0xfbfbfc),
	GroupStroke: rgb(0xd0d7de),
	NoteFill:    rgb(0xfff8c5),
	NoteStroke:  rgb(0xd4a72c),
	Background:  rgb(0xffffff),
	Highlight:   rgb(0xddf4ff),

	SectionFill:      rgba(0x54aeff, 26),
	SectionFill2:     rgba(0xd4a72c, 26),
	SectionAltFill:   rgba(0xffffff, 51),
	TaskFill:         rgb(0x80ccff),
	TaskStroke:       rgb(0x0969da),
	ActiveTaskFill:   rgb(0xddf4ff),
	ActiveTaskStroke: rgb(0x0969da),
	DoneTaskFill:     rgb(0xd0d7de),
	DoneTaskStroke:   rgb(0x8c959f),
	CritTaskFill:     rgb(0xffaba8),
	CritTaskStroke:   rgb(0xcf222e),
	TaskText:         rgb(0x1f2328),
	StatusTaskText:   rgb(0x1f2328),
	LinkText:         rgb(0x0550ae),
	VertLine:         rgb(0x8250df),
	TodayLine:        rgb(0xcf222e),
	ExcludeFill:      rgb(0xeaeef2),
	Tick:             rgba(0x1f2328, 204),
}

// Dark is sumi's own theme on a dark background.
var Dark = Palette{
	TextColor:   rgb(0xe6edf3),
	LineColor:   rgb(0x9198a1),
	NodeFill:    rgb(0x21262d),
	NodeStroke:  rgb(0x6e7681),
	GroupFill:   rgb(0x161b22),
	GroupStroke: rgb(0x3d444d),
	NoteFill:    rgb(0x3b2e0a),
	NoteStroke:  rgb(0x9e6a03),
	Background:  rgb(0x0d1117),
	Highlight:   rgb(0x132339),

	SectionFill:      rgba(0x388bfd, 26),
	SectionFill2:     rgba(0xbb8009, 26),
	SectionAltFill:   rgba(0x0d1117, 51),
	TaskFill:         rgb(0x1f6feb),
	TaskStroke:       rgb(0x58a6ff),
	ActiveTaskFill:   rgb(0x0c2d6b),
	ActiveTaskStroke: rgb(0x388bfd),
	DoneTaskFill:     rgb(0x30363d),
	DoneTaskStroke:   rgb(0x6e7681),
	CritTaskFill:     rgb(0x8e1519),
	CritTaskStroke:   rgb(0xf85149),
	TaskText:         rgb(0xffffff),
	StatusTaskText:   rgb(0xe6edf3),
	LinkText:         rgb(0xa5d6ff),
	VertLine:         rgb(0xa371f7),
	TodayLine:        rgb(0xf85149),
	ExcludeFill:      rgb(0x21262d),
	Tick:             rgba(0xe6edf3, 204),
}

// TextOn is black or white, whichever contrasts more with fill: for a label
// on a shape whose author set its fill but not its text's color. Black and
// white contrast equally with a fill of relative luminance 0.179.
func TextOn(fill Color) Color {
	if luminance(fill) > 0.179 {
		return Color{0, 0, 0, 255}
	}
	return Color{255, 255, 255, 255}
}

// luminance is c's relative luminance, as WCAG defines it, from 0 to 1.
func luminance(c Color) float64 {
	lin := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}
