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
	GetColor(float64) Color
	SetColor(Color)
}

type QuadSurface struct {
	V1, V2 Vector
	Color  Color
}

func SurfaceFromSurfaceData(sd SurfaceData) (res Surface, err error) {
	if len(sd) == 4 {
		qs := &QuadSurface{}
		qs.V1 = Vector{Start: sd[0], End: sd[1]}
		qs.V2 = Vector{Start: sd[0], End: sd[3]}
		res = qs
		return
	}
	err = errors.New(fmt.Sprintf("Got surface data with invalid number of poins: %d", len(sd)))
	return
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

func (qs *QuadSurface) GetColor(dist float64) (res Color) {
	logdist := math.Min(math.Log(dist)*45, 215.0)
	res = qs.Color
	res.R -= byte(logdist - 0.5)
	res.G -= byte(logdist)
	res.B -= byte(logdist + 0.5)
	return
}

func (qs *QuadSurface) SetColor(color Color) {
	qs.Color = color
}
