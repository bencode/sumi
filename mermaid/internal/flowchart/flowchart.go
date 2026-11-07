package flowchart

// Read reads the diagram in body, made ready by syntax.Prepare.
func Read(body string) (*Diagram, error) {
	ts, err := Lex(body)
	if err != nil {
		return nil, err
	}
	dir, stmts, err := Parse(ts)
	if err != nil {
		return nil, err
	}
	return Resolve(dir, stmts)
}
