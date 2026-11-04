// Package sequence lays out Mermaid sequence diagrams, drawing what
// Mermaid v12.3.0 draws for the syntax it supports. A diagram goes through
// a pipeline, each step taking values and returning new ones:
//
//	Lex      source -> tokens                        lex.go, modes.go
//	Parse    tokens -> a tree of statements          parse.go, signal.go
//	Resolve  the tree -> participants, boxes, numbers,
//	         activations checked                     resolve.go
//	measure, gaps, centers, measureAcross
//	         -> where things go across               measure.go, horizontal.go,
//	                                                 box.go
//	place    -> where things go down the page        place.go, activation.go,
//	                                                 box.go
//	draw     -> a scene                              draw.go, shapes.go, box.go
//
// The parser is a recursive descent over this grammar, taken from
// Mermaid's sequenceDiagram.jison with its left recursion written as
// repetition. Upper case names are tokens; REST is the rest of a line.
//
//	diagram     = { NEWLINE } "sequenceDiagram" document EOF ;
//	document    = { NEWLINE | statement } ;
//	statement   = participant | signal | note | activation | block | rect | box
//	            | "autonumber" [ "off" | NUM [ NUM ] ] NEWLINE
//	            | "title" REST NEWLINE
//	            | "accTitle" ":" REST NEWLINE
//	            | "accDescr" ( ":" REST | "{" TEXT "}" ) NEWLINE ;
//	participant = ( "participant" | "actor" ) ACTOR [ SETTINGS ] [ "as" REST ] NEWLINE ;
//	signal      = ACTOR [ "()" ] arrow [ "+" | "-" | "()" ] ACTOR TXT NEWLINE ;
//	note        = "note" ( ( "left of" | "right of" ) ACTOR
//	                     | "over" ACTOR [ "," ACTOR ] ) TXT NEWLINE ;
//	activation  = ( "activate" | "deactivate" ) ACTOR NEWLINE ;
//	block       = ( "loop" | "opt" | "break" ) REST document "end"
//	            | "alt"      REST document { "else"   REST document } "end"
//	            | "par"      REST document { "and"    REST document } "end"
//	            | "critical" REST document { "option" REST document } "end" ;
//	rect        = "rect" REST document "end" ;
//	box         = "box" REST { NEWLINE | participant } "end" ;
//	arrow       = "->" | "-->" | "->>" | "-->>" | "<<->>" | "<<-->>"
//	            | "-x" | "--x" | "-)" | "--)"
//	            | "-|\" | "-|/" | "-\\" | "-//" | "/|-" | "\|-" | "//-" | "\\-"
//	            | "--|\" | "--|/" | "--\\" | "--//" | "/|--" | "\|--" | "//--" | "\\--" ;
//
// Each production is a function that takes the tokens and returns what it
// read and the tokens left: Parse for diagram, parseDocument, parseStatement,
// parseParticipant, parseSignal, parseNote, parseActivation, parseAutonumber,
// parseBlock, parseRect and parseBox. Syntax Mermaid accepts that sumi does
// not yet, such as create, is refused with an error saying so. SETTINGS is
// @{...} after a participant's name: JSON for its type and its alias.
package sequence
