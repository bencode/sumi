package dagre

import (
	"slices"
	"testing"
)

func TestKeysOrder(t *testing.T) {
	var k keys[int]
	for _, n := range []string{"10", "b", "2", "01", "a", "4294967295", "1", "-1", "4294967294"} {
		k.set(n, 0)
	}
	k.del("b")
	k.set("b", 0)
	want := []string{"1", "2", "10", "4294967294", "01", "a", "4294967295", "-1", "b"}
	if got := k.names(); !slices.Equal(got, want) {
		t.Errorf("names() = %q, want %q", got, want)
	}
}
