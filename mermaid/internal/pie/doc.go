// Package pie reads and draws Mermaid pie charts.
//
//	lex     source -> tokens   lex.go
//	Read    tokens -> Diagram  read.go
//	Layout  Diagram -> Scene   draw.go
//
// Mermaid parses pie charts with Langium (pie.langium, Mermaid 12.3.0's parser package),
// whose lexer is Chevrotain's with rules Langium builds from the grammar: the keywords
// first, then the terminals the entry rule reaches, each place taking the first rule that
// matches. Some terminals the grammar defines are not among them. lex follows those rules;
// Read follows the grammar and what pieDb does with each section: a negative value is an
// error, and a label written again is passed over.
//
// Mermaid fails on any lexer or parser error; Read fails on the first, which is the first
// place Mermaid's error names. A < in a title or a description is not supported, as
// Mermaid sanitizes it.
//
// Layout draws what pieRenderer draws with the default config, which front matter and
// directives do not change here: the slices from 12 o'clock clockwise, the legend to the
// right, the percentages at 0.75 of the radius, and the slice of an empty label bigger, as
// the default highlightSlice is "". Its arcs are cubic curves. The colors are the
// theme's, and opaque, where Mermaid's slices are see-through. Numbers are written as Go
// writes them, which JavaScript writes alike but for those at or past 1e21 or below 1e-6,
// with an exponent there, and +Inf, which is Infinity there.
package pie
