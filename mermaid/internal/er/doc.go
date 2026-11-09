// Package er reads Mermaid entity relationship diagrams. A diagram goes through a
// pipeline, each step taking values and returning new ones:
//
//	Lex      source -> tokens       lex.go, rules.go
//	Parse    tokens -> statements   parse.go
//	collect  statements -> Diagram  collect.go
//	Size     Diagram -> entities    size.go
//	Place    Diagram -> Placement   place.go
//
// The lexer follows the rules Jison compiles from erDiagram.jison, which are not always
// those written there. The parser is a recursive descent over this grammar, from
// erDiagram.jison with its recursion written as repetition and the tokens the lexer never
// makes left out. Upper case names are tokens, and statements need nothing between them.
//
//	diagram    = ER_DIAGRAM { line } EOF ;
//	line       = statement | NEWLINE ;
//	statement  = relation | entity
//	           | ACC_TITLE ACC_TITLE_VALUE | ACC_DESCR ACC_DESCR_VALUE | ACC_MULTI_VALUE
//	           | DIRECTION
//	           | CLASSDEF idList styles separator
//	           | CLASS idList idList
//	           | STYLE idList styles separator
//	           | SUBGRAPH name [ SQS name { name } SQE ] separator { line } END ;
//	relation   = name [ classes ] cardinality ( IDENTIFYING | NON_IDENTIFYING ) cardinality
//	             name [ classes ] COLON role ;
//	entity     = name [ SQS name SQE ] [ classes ] [ BLOCK_START { attribute } BLOCK_STOP ] ;
//	classes    = STYLE_SEPARATOR idList ;
//	attribute  = ATTRIBUTE_WORD [ "?" ] ATTRIBUTE_WORD [ ATTRIBUTE_KEY { "," ATTRIBUTE_KEY } ] [ COMMENT ] ;
//	idList     = id { COMMA id } ;
//	id         = UNICODE_TEXT | STYLE_TEXT ;
//	styles     = style { COMMA style } ;
//	style      = ( STYLE_TEXT | COLON | BRKT ) { same } ;
//	separator  = NEWLINE | SEMI ;
//	name       = ENTITY_NAME | UNICODE_TEXT | NUM | DECIMAL_NUM | ENTITY_ONE ;
//	role       = WORD | ENTITY_NAME | UNICODE_TEXT ;
//	cardinality = ONLY_ONE | ZERO_OR_ONE | ZERO_OR_MORE | ONE_OR_MORE | MD_PARENT ;
//
// The grammar's separator also takes EOF and its line takes one too, but neither helps a
// source to be read: a separator's EOF leaves no EOF for the diagram to end with, and one
// inside a subgraph leaves its end unwritten.
package er
