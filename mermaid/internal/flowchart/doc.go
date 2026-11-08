// Package flowchart lays out and draws Mermaid flowcharts as Mermaid's neo look does,
// with ELK. A diagram goes through a pipeline, each step taking values and returning
// new ones:
//
//	Lex      source -> tokens              lex.go, rules.go, letters.go
//	Parse    tokens -> statements          parse.go, chain.go, click.go, arrow.go
//	Resolve  statements -> nodes and edges resolve.go, metadata.go, subgraph.go, data.go
//	Label    text -> lines of styled spans label.go, markdown.go
//	Box      lines -> wrapped lines, a box box.go, linebreak
//	Place    boxes -> shapes, a layout     layout.go, shape, graphlayout
//	Draw     a layout -> a scene           draw.go, edge.go, text.go
//
// Styles, classDef, style and linkStyle, are read and not yet drawn: every color is the
// theme's.
//
// The lexer follows the rules Jison compiles from flow.jison, which are not always
// those written there; rules.go says where. The parser is a recursive descent over
// this grammar, from flow.jison with its left recursion written as repetition and
// the tokens the lexer never makes left out. Upper case names are tokens.
//
//	diagram   = { SPACE | NEWLINE } GRAPH ( NODIR | DIR ( SEMI | { SPACE } NEWLINE ) ) document ;
//	document  = { statement | SEMI | NEWLINE | SPACE } ;
//	statement = chain sep
//	          | STYLE SPACE id SPACE styles sep
//	          | LINKSTYLE SPACE ( DEFAULT | nums ) SPACE
//	              ( styles | INTERPOLATE SPACE alnum [ SPACE styles ] ) sep
//	          | CLASSDEF SPACE id SPACE styles sep
//	          | CLASS SPACE id SPACE id sep
//	          | click sep
//	          | SUBGRAPH SPACE title [ SQS text SQE ] sep document END
//	          | DIRECTION
//	          | ACC_TITLE VALUE | ACC_DESCR VALUE | ACC_DESCR_BLOCK ;
//	sep       = NEWLINE | SEMI ;
//	chain     = node { link node } ;
//	node      = styled [ data ] { SPACE } { AMP SPACE { SPACE } styled [ data ] { SPACE } } ;
//	styled    = vertex [ STYLE_SEPARATOR id ] ;
//	vertex    = id [ shape ] ;
//	link      = [ LINK_ID ] LINK [ PIPE text PIPE [ SPACE ] ]
//	          | [ LINK_ID ] START_LINK edgetext LINK ;
//	data      = SHAPE_DATA { SHAPE_DATA } ;
//
// A & needs a SPACE before it as well, and shape is one of the fifteen brackets of
// chain.go. Mermaid's grammar also has a subgraph with no title, which crashes
// addSubGraph, so here it is an error.
//
// A label is read into what a browser shows of it: its entity codes, its line breaks, and
// of Markdown its paragraphs, emphasis and escapes. Mermaid gives labels to the browser as
// HTML, and all else that takes, icons, formulas and Markdown's blocks among them, sumi
// does not draw, so it is an error here; markdown.go lists it.
//
// Three things are read and dropped, as they change nothing Mermaid draws by default:
// linkStyle's interpolate and @{ curve } set a curve, which render.ts overrides on every
// edge with a route, and the length of a link is a hint only other layouts take.
package flowchart
