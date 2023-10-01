package goland

import (
	"math"
)

const (
	MIDNIGHT    = 0.0
	ONETHIRTY   = math.Pi * 0.25
	THREEO      = math.Pi * 0.5
	FOURTHIRTY  = math.Pi * 0.75
	SIXO        = math.Pi * 1.0
	SEVENTHIRTY = math.Pi * 1.25
	NINEO       = math.Pi * 1.5
	TENTHIRTY   = math.Pi * 1.75
	TWOPI       = math.Pi * 2.0
)

func AddAndNormalize(theta, freq float64) (res float64) {
	res = math.Mod(theta+freq+TWOPI, TWOPI)
	return
}
