package goland

import (
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

type Vector struct {
	Point
	Theta float64 `json:"theta"`
}

type Surface struct {
	P1 Point `json:"p1"`
	P2 Point `json:"p2"`
}

type Octree struct {
	Id                int
	SurfaceIndices    []int
	Surfaces          []Surface
	Children          []Octree
	Parent            *Octree
	XMin, XMax, XAxis float64
	YMin, YMax, YAxis float64
	ZMin, ZMax, ZAxis float64
}

// see: https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection
func segmentIntersection(s1start, s1end, s2start, s2end Point) (intersection Point) {

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

func (tree *Octree) getSurfacesInTree(surfaces []Surface) (res []int) {
	res = make([]int, 0)
	for ix, surface := range surfaces {
		if tree.XMin <= surface.P1.X && surface.P1.X <= tree.XMax && tree.YMin <= surface.P1.Y && surface.P1.Y <= tree.YMax {
			res = append(res, ix)
			continue
		}
		if tree.XMin <= surface.P2.X && surface.P2.X <= tree.XMax && tree.YMin <= surface.P2.Y && surface.P2.Y <= tree.YMax {
			res = append(res, ix)
			continue
		}
		boundary1 := tree.LLBoundary()
		boundary2 := tree.ULBoundary()
		intersection := segmentIntersection(surface.P1, surface.P2, boundary1, boundary2)
		if !math.IsNaN(intersection.X) {
			res = append(res, ix)
			continue
		}
		boundary1 = tree.ULBoundary()
		boundary2 = tree.URBoundary()
		intersection = segmentIntersection(surface.P1, surface.P2, boundary1, boundary2)
		if !math.IsNaN(intersection.X) {
			res = append(res, ix)
			continue
		}
		boundary1 = tree.URBoundary()
		boundary2 = tree.LRBoundary()
		intersection = segmentIntersection(surface.P1, surface.P2, boundary1, boundary2)
		if !math.IsNaN(intersection.X) {
			res = append(res, ix)
			continue
		}
		boundary1 = tree.LRBoundary()
		boundary2 = tree.LLBoundary()
		intersection = segmentIntersection(surface.P1, surface.P2, boundary1, boundary2)
		if !math.IsNaN(intersection.X) {
			res = append(res, ix)
			continue
		}
	}
	return
}

func constructOctree(tree *Octree, surfaces []Surface) (err error) {

	tree.Id = TREEID
	TREEID++

	tree.XAxis = (tree.XMin + tree.XMax) / 2
	tree.YAxis = (tree.YMin + tree.YMax) / 2

	tree.SurfaceIndices = tree.getSurfacesInTree(surfaces)

	// no need to recurse if small number of surfaces
	if len(tree.SurfaceIndices) <= 1 {
		return
	}

	// check for min size
	if tree.XMax-tree.XMin <= MINOCTREESIZE {
		return
	}

	if tree.YMax-tree.YMin <= MINOCTREESIZE {
		return
	}

	tree.Children = make([]Octree, 4)

	tree.Children[0].Parent = tree
	tree.Children[0].XMin = tree.XAxis
	tree.Children[0].XMax = tree.XMax
	tree.Children[0].YMin = tree.YMin
	tree.Children[0].YMax = tree.YAxis

	tree.Children[1].Parent = tree
	tree.Children[1].XMin = tree.XAxis
	tree.Children[1].XMax = tree.XMax
	tree.Children[1].YMin = tree.YAxis
	tree.Children[1].YMax = tree.YMax

	tree.Children[2].Parent = tree
	tree.Children[2].XMin = tree.XMin
	tree.Children[2].XMax = tree.XAxis
	tree.Children[2].YMin = tree.YAxis
	tree.Children[2].YMax = tree.YMax

	tree.Children[3].Parent = tree
	tree.Children[3].XMin = tree.XMin
	tree.Children[3].XMax = tree.XAxis
	tree.Children[3].YMin = tree.YMin
	tree.Children[3].YMax = tree.YAxis

	for i := 0; i < 4; i++ {
		err = constructOctree(&tree.Children[i], surfaces)
		if err != nil {
			fmt.Printf("Got err from constructOctree: %s\n", err)
			return
		}
	}

	return

}

func NewOctreeFromSurfaces(surfaces []Surface) (root *Octree, err error) {

	root = &Octree{}
	root.SurfaceIndices = make([]int, len(surfaces))
	for i := 0; i < len(surfaces); i++ {
		root.SurfaceIndices[i] = i
	}
	root.Surfaces = surfaces
	root.XMin = -1024
	root.XMax = 1024
	root.YMin = -1024
	root.YMax = 1024

	err = constructOctree(root, surfaces)
	if err != nil {
		fmt.Printf("Got error constructing octree!\n")
		root = nil
	}

	return

}

func (root *Octree) getTreeByPoint(point Point) (res *Octree) {

	current := root

	if point.X >= root.XMax || point.X < root.XMin {
		return
	}

	if point.Y >= root.YMax || point.Y < root.YMin {
		return
	}

	for {
		if current.Children == nil {
			break
		}
		if point.X >= current.XAxis {
			if point.Y < current.YAxis {
				current = &current.Children[0]
				continue
			} else {
				current = &current.Children[1]
				continue
			}
		} else {
			if point.Y >= current.YAxis {
				current = &current.Children[2]
				continue
			} else {
				current = &current.Children[3]
				continue
			}
		}
	}

	res = current
	return

}

func (tree *Octree) innerFindSurface(ogstart, s1end Point, facingIx int) (endpoint Point, dist float64) {
	endpoint.X = math.NaN()
	endpoint.Y = math.NaN()
	endpoint.Z = math.NaN()
	var s1start, s2start, s2end Point
	s1start = ogstart
	surfaces := tree.Surfaces
	closestDist2 := math.Pow(2048, 2)
	for {
		subtree := tree.getTreeByPoint(s1start)
		if subtree == nil {
			return
		}
		for _, surfaceIx := range subtree.SurfaceIndices {
			intersection := segmentIntersection(ogstart, s1end, surfaces[surfaceIx].P1, surfaces[surfaceIx].P2)
			if !math.IsNaN(intersection.X) {
				currentDist2 := getDist2(ogstart, intersection)
				if currentDist2 <= closestDist2 {
					endpoint = intersection
					closestDist2 = currentDist2
				}
			}
		}
		// Because surfaces can span multiple trees,
		// it's possible we have an endpoint which is not the solution
		if !math.IsNaN(endpoint.X) && subtree.Id == tree.getTreeByPoint(endpoint).Id {
			dist = math.Sqrt(closestDist2)
			return
		}
		doContinue := false
		for i := 0; i < 4; i++ {
			switch (facingIx + i) % 4 {
			case 0:
				s2start = subtree.LLBoundary()
				s2end = subtree.ULBoundary()
			case 1:
				s2start = subtree.ULBoundary()
				s2end = subtree.URBoundary()
			case 2:
				s2start = subtree.URBoundary()
				s2end = subtree.LRBoundary()
			case 3:
				s2start = subtree.LRBoundary()
				s2end = subtree.LLBoundary()
			default:
				endpoint.X = math.NaN()
				endpoint.Y = endpoint.X
				endpoint.Z = endpoint.X
				return
			}
			intersection := segmentIntersection(s1start, s1end, s2start, s2end)
			if !math.IsNaN(intersection.X) {
				doContinue = true
				s1start = intersection
				break
			}
		}
		if !doContinue {
			//fmt.Println("Warning: doSingleRay could not find boundary\n")
			endpoint.X = math.NaN()
			endpoint.Y = endpoint.X
			endpoint.Z = endpoint.X
			return
		}
	}
}

func (tree *Octree) TraceVector(vec Vector, maxDist float64) (Point, float64) {
	s1start := vec.Point
	advanced := Advance(vec, maxDist)
	s1end := advanced.Point
	facingIx := 0
	if vec.Theta < THREEO {
		facingIx = 1
	} else if vec.Theta < SIXO {
		facingIx = 2
	} else if vec.Theta < NINEO {
		facingIx = 3
	}
	return tree.innerFindSurface(s1start, s1end, facingIx)
}

func (tree *Octree) TraceSegment(s1start, s1end Point) (Point, float64) {
	xp := s1end.X - s1start.X
	yp := s1end.Y - s1start.Y
	facingIx := 0
	if xp >= 0 {
		if yp < 0 {
			facingIx = 1
		} else {
			facingIx = 2
		}
	} else {
		if yp >= 0 {
			facingIx = 3
		} else {
			facingIx = 0
		}
	}
	return tree.innerFindSurface(s1start, s1end, facingIx)
}

func (tree *Octree) ULBoundary() Point {
	return Point{
		X: tree.XMin - MILLI,
		Y: tree.YMin - MILLI,
	}
}

func (tree *Octree) URBoundary() Point {
	return Point{
		X: tree.XMax + MILLI,
		Y: tree.YMin - MILLI,
	}
}

func (tree *Octree) LRBoundary() Point {
	return Point{
		X: tree.XMax + MILLI,
		Y: tree.YMax + MILLI,
	}
}

func (tree *Octree) LLBoundary() Point {
	return Point{
		X: tree.XMin - MILLI,
		Y: tree.YMax + MILLI,
	}
}

func getDist(p1, p2 Point) float64 {
	xx := float64(p1.X - p2.X)
	yy := float64(p1.Y - p2.Y)
	zz := float64(p1.Z - p2.Z)
	return math.Sqrt(xx*xx + yy*yy + zz*zz)
}

func getDist2(p1, p2 Point) float64 {
	xx := float64(p1.X - p2.X)
	yy := float64(p1.Y - p2.Y)
	zz := float64(p1.Z - p2.Z)
	return xx*xx + yy*yy + zz*zz
}
