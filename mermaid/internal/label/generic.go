package label

import (
	"bytes"
	"strings"
)

// GenericTypes is parseGenericTypes: the ~ of a generic become < and >, across the
// comma when each side has one.
func GenericTypes(input string) string {
	sets := splitCommas(input)
	var out []string
	for i := 0; i < len(sets); i++ {
		set := sets[i]
		if set == "," && i > 0 && i+1 < len(sets) {
			if prev, next := sets[i-1], sets[i+1]; strings.Count(prev, "~") == 1 && strings.Count(next, "~") == 1 {
				set = prev + "," + next
				i++
				out = out[:len(out)-1]
			}
		}
		out = append(out, processSet(set))
	}
	return strings.Join(out, "")
}

// splitCommas is input.split(/(,)/): the parts between commas and the commas.
func splitCommas(input string) []string {
	var out []string
	for i, part := range strings.Split(input, ",") {
		if i > 0 {
			out = append(out, ",")
		}
		out = append(out, part)
	}
	return out
}

// processSet pairs a set's ~ from the outside in; an odd one at its start stays.
func processSet(input string) string {
	n := strings.Count(input, "~")
	if n <= 1 {
		return input
	}
	starting := n%2 != 0 && strings.HasPrefix(input, "~")
	if starting {
		input = input[1:]
	}
	b := []byte(input)
	for first, last := bytes.IndexByte(b, '~'), bytes.LastIndexByte(b, '~'); first != last; first, last = bytes.IndexByte(b, '~'), bytes.LastIndexByte(b, '~') {
		b[first], b[last] = '<', '>'
	}
	if starting {
		return "~" + string(b)
	}
	return string(b)
}
