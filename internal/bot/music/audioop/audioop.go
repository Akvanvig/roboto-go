package audioop

import (
	"math"
)

const (
	MinVal = -0x8000
	MaxVal = 0x7fff
)

func fBound(val float64) float64 {
	if val > MaxVal {
		val = MaxVal
	} else if val < (MinVal + 1.0) {
		val = MinVal
	}

	return math.Floor(val)
}

// Note(Fredrico):
// Contemplate making this into a generic
func Mul(fragment []int16, factor float64) {
	for i, val := range fragment {
		fragment[i] = int16(fBound(float64(val) * factor))
	}
}
