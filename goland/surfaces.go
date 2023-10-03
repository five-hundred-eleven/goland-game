package goland

import (
	"errors"
	"fmt"
	"math"
)

const (
	MINSURFACES   = 2
	MINOCTREESIZE = 8
)

var TREEID = 0

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Color struct {
	R, G, B byte
}

type Vector struct {
	Start, End Point
}

type Surface interface {
	GetIntersection(Vector) Point
	GetPoints() []Point
	GetColor(Point) Color
	SetColor(Color)
}

type SegmentSurface struct {
	Vector
	Color Color
}

type QuadSurface struct {
	V1, V2 Vector
	Color  Color
}

func SurfaceFromSurfaceData(sd SurfaceData) (res Surface, err error) {
	if len(sd) == 4 {
		/*
			qs := &QuadSurface{}
			qs.V1 = Vector{Start: sd[0], End: sd[1]}
			qs.V2 = Vector{Start: sd[0], End: sd[3]}
			res = qs
			return
		*/
		s := &SegmentSurface{
			Vector: Vector{
				Start: sd[0],
				End:   sd[1],
			},
			Color: Color{
				R: 255,
				G: 255,
				B: 255,
			},
		}
		res = s
		return
	}
	err = errors.New(fmt.Sprintf("Got surface data with invalid number of poins: %d", len(sd)))
	return
}

// see: https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection
func (s *SegmentSurface) GetIntersection(vec Vector) (intersection Point) {

	d := (s.Start.X-s.End.X)*(vec.Start.Y-vec.End.Y) - (s.Start.Y-s.End.Y)*(vec.Start.X-vec.End.X)

	if math.Abs(d) < 1e-6 {
		intersection.X = math.NaN()
		intersection.Y = intersection.X
		return
	}

	tn := (s.Start.X-vec.Start.X)*(vec.Start.Y-vec.End.Y) - (s.Start.Y-vec.Start.Y)*(vec.Start.X-vec.End.X)
	if math.Signbit(d) != math.Signbit(tn) || math.Abs(d) < math.Abs(tn) {
		intersection.X = math.NaN()
		intersection.Y = intersection.X
		return
	}
	un := (s.Start.X-vec.Start.X)*(s.Start.Y-s.End.Y) - (s.Start.Y-vec.Start.Y)*(s.Start.X-s.End.X)
	if math.Signbit(d) != math.Signbit(un) || math.Abs(d) < math.Abs(un) {
		intersection.X = math.NaN()
		intersection.Y = intersection.X
		return
	}

	intersection = Point{}
	intersection.X = ((s.Start.X*s.End.Y-s.Start.Y*s.End.X)*(vec.Start.X-vec.End.X) - (s.Start.X-s.End.X)*(vec.Start.X*vec.End.Y-vec.Start.Y*vec.End.X)) / d
	intersection.Y = ((s.Start.X*s.End.Y-s.Start.Y*s.End.X)*(vec.Start.Y-vec.End.Y) - (s.Start.Y-s.End.Y)*(vec.Start.X*vec.End.Y-vec.Start.Y*vec.End.X)) / d

	return

}

func (s *SegmentSurface) GetPoints() (points []Point) {
	points = []Point{s.Start, s.End}
	return
}

func (s *SegmentSurface) GetColor(relativePoint Point) (res Color) {
	if relativePoint.Z > WALLHEIGHT {
		res.R = 0
		res.G = 0
		res.B = 0
		return
	}
	if relativePoint.Z < FLOORHEIGHT {
		theta := math.Atan(relativePoint.Y / relativePoint.X)
		floorDist := math.Abs(FLOORHEIGHT) / math.Tan(theta)
		shade := 34 - math.Log(floorDist)*6
		res.R = byte(shade + 0.5)
		res.G = byte(shade)
		res.B = byte(shade - 0.5)
		return
	}
	dist := relativePoint.X*relativePoint.X + relativePoint.Y*relativePoint.Y
	dist = math.Sqrt(dist)
	dist = math.Min(math.Log(dist)*45, 215.0)
	res = s.Color
	res.R -= byte(dist - 0.5)
	res.G -= byte(dist)
	res.B -= byte(dist + 0.5)
	return
}

func (s *SegmentSurface) SetColor(color Color) {
	s.Color = color
}

func (qs *QuadSurface) GetIntersection(vec Vector) (res Point) {

	res.X = math.NaN()
	res.Y = math.NaN()
	res.Z = math.NaN()

	lba := VectorSub(vec.Start, vec.End)
	v01 := VectorSub(qs.V1.End, qs.V1.Start)
	v02 := VectorSub(qs.V2.End, qs.V2.Start)
	cross12 := VectorCross(v01, v02)
	det := VectorDot(lba, cross12)
	if math.Abs(det) < MICRO {
		return
	}

	diffStart := VectorSub(vec.Start, qs.V1.Start)
	tn := VectorDot(cross12, diffStart)
	if math.Signbit(tn) != math.Signbit(det) || math.Abs(det) < math.Abs(tn) {
		return
	}
	coefU := VectorCross(v02, lba)
	un := VectorDot(coefU, diffStart)
	if math.Signbit(un) != math.Signbit(det) || math.Abs(det) < math.Abs(un) {
		return
	}
	coefV := VectorCross(lba, v01)
	vn := VectorDot(coefV, diffStart)
	if math.Signbit(vn) != math.Signbit(det) || math.Abs(det) < math.Abs(vn) {
		return
	}

	lab := VectorSub(vec.End, vec.Start)
	t := tn / det

	res = Point{
		X: vec.Start.X + lab.X*t,
		Y: vec.Start.Y + lab.Y*t,
		Z: vec.Start.Z + lab.Z*t,
	}

	return

}

func (qs *QuadSurface) GetPoints() (res []Point) {
	res = make([]Point, 4)
	res[0] = qs.V1.Start
	res[1] = qs.V1.End
	res[2] = VectorAdd(qs.V1.End, VectorSub(qs.V2.End, qs.V2.Start))
	res[3] = qs.V2.End
	return
}

func (qs *QuadSurface) GetColor(relativePoint Point) (res Color) {
	res = qs.Color
	return
}

func (qs *QuadSurface) SetColor(color Color) {
	qs.Color = color
}
