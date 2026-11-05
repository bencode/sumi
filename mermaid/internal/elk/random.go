// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

const (
	randomMultiplier = 0x5DEECE66D
	randomMask       = 1<<48 - 1
)

// random is java.util.Random, the generator ELK takes its random choices from.
type random struct{ seed uint64 }

func newRandom(seed int64) *random {
	r := &random{}
	r.setSeed(seed)
	return r
}

func (r *random) setSeed(seed int64) {
	r.seed = (uint64(seed) ^ randomMultiplier) & randomMask
}

// next gives the top bits of the next state, signed when that is all 32.
func (r *random) next(bits int) int32 {
	r.seed = (r.seed*randomMultiplier + 0xB) & randomMask
	return int32(r.seed >> (48 - bits))
}

// nextInt follows elkjs, not Java: its numbers are doubles, so the retry on
// overflow never happens.
func (r *random) nextInt(n int32) int32 {
	bits := r.next(31)
	if n&-n == n {
		return int32(int64(n) * int64(bits) >> 31)
	}
	return bits % n
}

func (r *random) nextBoolean() bool { return r.next(1) != 0 }

// nextFloat is a float64: elkjs runs as JavaScript, where a float is a double.
func (r *random) nextFloat() float64 { return float64(r.next(24)) / (1 << 24) }

func (r *random) nextDouble() float64 {
	high := float64(r.next(26)) / (1 << 26)
	return high + float64(r.next(27))/(1<<53)
}

func (r *random) nextLong() int64 {
	high := int64(r.next(32))
	return high<<32 + int64(r.next(32))
}
