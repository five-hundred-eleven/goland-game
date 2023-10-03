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

func GetDist(p1, p2 Point) float64 {
	xx := float64(p1.X - p2.X)
	yy := float64(p1.Y - p2.Y)
	zz := float64(p1.Z - p2.Z)
	return math.Sqrt(xx*xx + yy*yy + zz*zz)
}

func GetDist2(p1, p2 Point) float64 {
	xx := float64(p1.X - p2.X)
	yy := float64(p1.Y - p2.Y)
	zz := float64(p1.Z - p2.Z)
	return xx*xx + yy*yy + zz*zz
}

func GetRelativePoint(p1, p2 Point) (res Point) {
	res.X = p2.X - p1.X
	res.Y = p2.Y - p1.Y
	res.Z = p2.Z - p1.Z
	return
}

func VectorCross(a, b Point) (res Point) {
	res.X = a.Y*b.Z - a.Z*b.Y
	res.Y = a.Z*b.X - a.X*b.Z
	res.Z = a.X*b.Y - a.Y*b.X
	return
}

func VectorDot(a, b Point) (res float64) {
	res = a.X*b.X + a.Y*b.Y + a.Z*b.Z
	return
}

func VectorAdd(a, b Point) (res Point) {
	res.X = a.X + b.X
	res.Y = a.Y + b.Y
	res.Z = a.Z + b.Z
	return
}

func VectorSub(a, b Point) (res Point) {
	res.X = a.X - b.X
	res.Y = a.Y - b.Y
	res.Z = a.Z - b.Z
	return
}

// see: https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection
func SegmentIntersection(s1start, s1end, s2start, s2end Point) (intersection Point) {

	d := (s1start.X-s1end.X)*(s2start.Y-s2end.Y) - (s1start.Y-s1end.Y)*(s2start.X-s2end.X)

	if math.Abs(d) < 1e-6 {
		intersection.X = math.NaN()
		intersection.Y = intersection.X
		return
	}

	tn := (s1start.X-s2start.X)*(s2start.Y-s2end.Y) - (s1start.Y-s2start.Y)*(s2start.X-s2end.X)
	if math.Signbit(d) != math.Signbit(tn) || math.Abs(d) < math.Abs(tn) {
		intersection.X = math.NaN()
		intersection.Y = intersection.X
		return
	}
	un := (s1start.X-s2start.X)*(s1start.Y-s1end.Y) - (s1start.Y-s2start.Y)*(s1start.X-s1end.X)
	if math.Signbit(d) != math.Signbit(un) || math.Abs(d) < math.Abs(un) {
		intersection.X = math.NaN()
		intersection.Y = intersection.X
		return
	}

	intersection = Point{}
	intersection.X = ((s1start.X*s1end.Y-s1start.Y*s1end.X)*(s2start.X-s2end.X) - (s1start.X-s1end.X)*(s2start.X*s2end.Y-s2start.Y*s2end.X)) / d
	intersection.Y = ((s1start.X*s1end.Y-s1start.Y*s1end.X)*(s2start.Y-s2end.Y) - (s1start.Y-s1end.Y)*(s2start.X*s2end.Y-s2start.Y*s2end.X)) / d

	return

}
