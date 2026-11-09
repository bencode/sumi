// Package class reads Mermaid class diagrams. A diagram goes through a pipeline, each
// step taking values and returning new ones:
//
//	Lex      source -> tokens       lex.go, rules.go
//	Parse    tokens -> statements   parse.go
//	Collect  statements -> Diagram  collect.go, member.go
//
// The lexer follows the rules Jison compiles from classDiagram.jison, which are not
// always those written there. The parser is a recursive descent over this grammar,
// from classDiagram.jison with its recursion written as repetition and the tokens the
// lexer never makes left out. Upper case names are tokens.
//
//	diagram    = CLASS_DIAGRAM NL statements END ;
//	statements = statement { NL statement } [ NL ] ;
//	statement  = relation [ LABEL ]
//	           | className [ LABEL ]
//	           | namespace | class | note
//	           | ANNOTATION_START alnum ANNOTATION_END className
//	           | CALLBACK className STRING [ STRING ]
//	           | LINK className STRING link
//	           | CLICK className ( CALLBACK_NAME [ CALLBACK_ARGS ] [ STRING ] | HREF STRING link )
//	           | STYLE ALPHA styles
//	           | CSS_CLASS STRING ALPHA
//	           | CLASS_DEF ALPHA { COMMA ALPHA } styles
//	           | DIRECTION
//	           | ACC_TITLE ACC_TITLE_VALUE | ACC_DESCR ACC_DESCR_VALUE | ACC_BLOCK_VALUE ;
//	link       = [ LINK_TARGET | STRING [ LINK_TARGET ] ] ;
//	styles     = style { COMMA style } ;
//	style      = ( ALPHA | COLON | BRKT | STYLE | PCT | LABEL ) { same } ;
//	namespace  = NAMESPACE namespaceName [ classLabel ] STRUCT_START [ NL ] items STRUCT_STOP ;
//	items      = item { NL item } [ NL ] ;
//	item       = class | note | namespace ;
//	class      = CLASS className [ classLabel ] [ STYLE_SEPARATOR alnum | annotation ] [ body ] ;
//	annotation = ANNOTATION_START alnum ANNOTATION_END ;
//	body       = STRUCT_START { MEMBER } STRUCT_STOP ;     (one MEMBER at least after a STYLE_SEPARATOR)
//	note       = NOTE_FOR className STRING | NOTE STRING ;
//	relation   = className [ STRING ] arrow [ STRING ] className ;
//	arrow      = [ end ] line [ end ] ;
//	line       = LINE | DOTTED_LINE ;
//	end        = AGGREGATION | EXTENSION | COMPOSITION | DEPENDENCY | LOLLIPOP ;
//	classLabel = SQUARE_OPEN STRING SQUARE_CLOSE ;
//	className  = alnum [ DOT className | GENERIC | className ] | BACKQUOTE [ GENERIC ] ;
//	namespaceName = alnum [ [ DOT ] namespaceName ] | BACKQUOTE ;
//	alnum      = UNICODE | ALPHA | MINUS ;
package class
