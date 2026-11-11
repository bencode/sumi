// Package pie reads Mermaid pie charts.
//
//	lex   source -> tokens   lex.go
//	Read  tokens -> Diagram  read.go
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
package pie
