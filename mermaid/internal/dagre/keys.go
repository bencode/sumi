package dagre

import (
	"cmp"
	"slices"
	"strconv"
	"unicode/utf16"
)

// keys are a JS object's properties, listed in the order JS lists them:
// array indices ("0", "42", but not "01", and below 2^32-1) by value
// first, then the rest in the order they were added. Deleting a key and
// setting it again moves it to the end. graphlib keeps all its nodes and
// edges in such objects, so this order is what dagre's ties go by.
type keys[V any] struct {
	m       map[string]V
	order   []string // the order the keys were added in
	indices int      // how many of them are array indices
}

func (k *keys[V]) set(name string, v V) {
	if k.m == nil {
		k.m = map[string]V{}
	}
	if _, ok := k.m[name]; !ok {
		k.order = append(k.order, name)
		if isIndex(name) {
			k.indices++
		}
	}
	k.m[name] = v
}

func (k *keys[V]) get(name string) (V, bool) {
	v, ok := k.m[name]
	return v, ok
}

func (k *keys[V]) has(name string) bool {
	_, ok := k.m[name]
	return ok
}

func (k *keys[V]) del(name string) {
	if !k.has(name) {
		return
	}
	delete(k.m, name)
	i := slices.Index(k.order, name)
	k.order = slices.Delete(k.order, i, i+1)
	if isIndex(name) {
		k.indices--
	}
}

func (k *keys[V]) len() int { return len(k.m) }

// names are the keys in JS order, in a slice of their own, so the caller
// may change k while going through them.
func (k *keys[V]) names() []string {
	out := slices.Clone(k.order)
	if k.indices > 0 {
		slices.SortStableFunc(out, func(a, b string) int {
			ia, ib := isIndex(a), isIndex(b)
			if ia && ib {
				return cmp.Compare(index(a), index(b))
			}
			return cmp.Compare(boolInt(ib), boolInt(ia))
		})
	}
	return out
}

// values are the values, in names' order.
func (k *keys[V]) values() []V {
	names := k.names()
	out := make([]V, len(names))
	for i, n := range names {
		out[i] = k.m[n]
	}
	return out
}

// isIndex is whether s is an array index: the canonical decimal form of
// an integer from 0 to 2^32-2 (ECMAScript's CanonicalNumericIndexString).
func isIndex(s string) bool {
	if s == "" || len(s) > 1 && s[0] == '0' {
		return false
	}
	n, err := strconv.ParseUint(s, 10, 32)
	return err == nil && n < 1<<32-1
}

func index(s string) uint64 {
	n, _ := strconv.ParseUint(s, 10, 32) // s is an index
	return n
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// jsLess is a < b as JS compares strings: by UTF-16 code units, which
// sorts characters past U+FFFF before U+E000 to U+FFFF, unlike Go.
func jsLess(a, b string) bool {
	return slices.Compare(utf16.Encode([]rune(a)), utf16.Encode([]rune(b))) < 0
}
