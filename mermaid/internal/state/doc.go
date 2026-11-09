// Package state reads Mermaid state diagrams. A diagram goes through a pipeline, each
// step taking values and returning new ones:
//
//	Lex        source -> tokens         lex.go, rules.go
//	Parse      tokens -> statements     parse.go
//	Translate  statements -> statements translate.go
//	Extract    statements -> tables     extract.go
//	Fetch      statements -> a diagram  fetch.go, data.go
//	Size       a diagram -> sizes       size.go
//	Place      sizes -> a layout        layout.go, graphlayout
//
// Translate runs twice, as Mermaid's parsing and drawing each run it; Read says why.
// The lexer follows the rules Jison compiles from stateDiagram.jison, which are not
// always those written there. The parser is a recursive descent over this grammar,
// from stateDiagram.jison with its left recursion written as repetition and the
// tokens the lexer never makes left out. Upper case names are tokens.
//
//	diagram   = { NL } SD document ;
//	document  = { statement | NL } ;
//	statement = CLASSDEF CLASSDEF_ID CLASSDEF_STYLES
//	          | STYLE STYLE_IDS STYLE_STYLES
//	          | CLASS CLASS_IDS CLASS_NAME
//	          | id [ DESCR ]
//	          | id ARROW id [ DESCR ]
//	          | HIDE_EMPTY | SCALE WIDTH
//	          | COMPOSITE [ struct ]
//	          | STATE_DESCR AS ID [ struct ]
//	          | FORK | JOIN | CHOICE | CONCURRENT
//	          | NOTE ( LEFT_OF | RIGHT_OF ) ID NOTE_TEXT
//	          | NOTE NOTE_TEXT AS ID
//	          | DIRECTION
//	          | ACC_TITLE ACC_TITLE_VALUE | ACC_DESCR ACC_DESCR_VALUE | ACC_BLOCK_VALUE
//	          | CLICK id ( STRING STRING | HREF STRING ) NL ;
//	struct    = STRUCT_START document STRUCT_STOP ;
//	id        = ( ID | EDGE_STATE ) [ STYLE_SEP ID ] ;
package state
