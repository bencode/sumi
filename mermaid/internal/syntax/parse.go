package syntax

// Accept takes the first token of ts if it is of kind k.
func Accept(ts []Token, k Kind) (Token, []Token, bool) {
	if ts[0].Kind != k {
		return Token{}, ts, false
	}
	return ts[0], ts[1:], true
}

// Expect takes the first token of ts, which must be of kind k; want names k
// in the error otherwise. ts ends with an EOF token, which Expect never takes
// unless k is EOF.
func Expect(ts []Token, k Kind, want string) (Token, []Token, error) {
	t, rest, ok := Accept(ts, k)
	if !ok {
		return t, ts, Unexpected(ts[0], want)
	}
	return t, rest, nil
}

// Unexpected is the error for finding t where want was expected.
func Unexpected(t Token, want string) *Error {
	return Errorf(t.Pos, "unexpected %s, expected %s", Describe(t), want)
}

// Describe names t for an error.
func Describe(t Token) string {
	if t.Kind == EOF {
		return "end of input"
	}
	if t.Kind == Newline && t.Text != ";" {
		return "end of line"
	}
	return `"` + t.Text + `"`
}
