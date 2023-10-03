package goland

import (
	"fmt"
	"math"
	"sync"
)

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

type SurfaceCache struct {
	SurfaceIndices []int
	Dists2         []float64
	Visited        sync.Map
}

func NewSurfaceCache() (cache *SurfaceCache) {
	cache = &SurfaceCache{
		SurfaceIndices: make([]int, 0, 16),
		Dists2:         make([]float64, 0, 16),
	}
	return
}

func (cache *SurfaceCache) Bisect(dist2 float64) (mid int) {
	lo := 0
	mid = 0
	hi := len(cache.Dists2)
	for lo < hi {
		mid = (lo + hi) / 2
		if dist2 > cache.Dists2[mid] {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return
}

func (cache *SurfaceCache) Insort(ix int, dist2 float64) {
	mid := cache.Bisect(dist2)
	cache.SurfaceIndices = append(cache.SurfaceIndices, 0)
	cache.Dists2 = append(cache.Dists2, 0.0)
	copy(cache.SurfaceIndices[mid+1:], cache.SurfaceIndices[mid:])
	cache.SurfaceIndices[mid] = ix
	copy(cache.Dists2[mid+1:], cache.Dists2[mid:])
	cache.Dists2[mid] = dist2
	cache.Visited.Store(ix, true)
}

func (cache *SurfaceCache) Contains(ix int) bool {
	_, isOk := cache.Visited.Load(ix)
	return isOk
}

func (tree *Octree) getSurfacesInTree(surfaces []Surface, parentSurfaceIndices []int) (res []int) {
	res = make([]int, 0)
	for _, ix := range parentSurfaceIndices {
		surface := surfaces[ix]
		points := surface.GetPoints()
		doContinue := false
		for _, point := range points {
			if tree.XMin <= point.X && point.X <= tree.XMax && tree.YMin <= point.Y && point.Y <= tree.YMax { //&& tree.ZMin <= point.Z && point.Z <= tree.ZMax {
				/*
					if tree.Id != 0 {
						fmt.Printf("Storing surface in tree by point: %d: (%f, %f, %f)\n", tree.Id, point.X, point.Y, point.Z)
						fmt.Printf("Tree X: (%f, %f)\n", tree.XMin, tree.XMax)
						fmt.Printf("Tree Y: (%f, %f)\n", tree.YMin, tree.YMax)
						fmt.Printf("Tree Z: (%f, %f)\n", tree.ZMin, tree.ZMax)
					}
				*/
				res = append(res, ix)
				doContinue = true
				break
			}
		}
		if doContinue {
			continue
		}
		if len(points) == 1 {
			continue
		}
		for jx := 0; jx < len(points); jx++ {
			kx := (jx + 1) % len(points)
			vec := Vector{
				Start: points[jx],
				End:   points[kx],
			}
			neighbor := tree.testBoundaries(vec, 0)
			if !math.IsNaN(neighbor.X) {
				//fmt.Printf("Storing surface in tree by intersection: %d\n", tree.Id)
				res = append(res, ix)
				break
			}
		}
	}
	return
}

func constructOctree(tree *Octree, surfaces []Surface, parentSurfaceIndices []int) (err error) {

	tree.Id = TREEID
	TREEID++

	tree.XAxis = (tree.XMin + tree.XMax) / 2
	tree.YAxis = (tree.YMin + tree.YMax) / 2
	tree.ZAxis = (tree.ZMin + tree.ZMax) / 2

	tree.SurfaceIndices = tree.getSurfacesInTree(surfaces, parentSurfaceIndices)

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

	if tree.ZMax-tree.ZMin <= MINOCTREESIZE {
		return
	}

	tree.Children = make([]Octree, 8)

	for i := 0; i < 8; i++ {
		tree.Children[i].Parent = tree
		if i&2 == 0 {
			tree.Children[i].XMin = tree.XAxis
			tree.Children[i].XMax = tree.XMax
			if i&1 == 0 {
				tree.Children[i].YMin = tree.YMin
				tree.Children[i].YMax = tree.YAxis
			} else {
				//fmt.Printf("Constructing top right quad tree\n")
				tree.Children[i].YMin = tree.YAxis
				tree.Children[i].YMax = tree.YMax
			}
		} else {
			tree.Children[i].XMin = tree.XMin
			tree.Children[i].XMax = tree.XAxis
			if i&1 == 0 {
				tree.Children[i].YMin = tree.YAxis
				tree.Children[i].YMax = tree.YMax
			} else {
				tree.Children[i].YMin = tree.YMin
				tree.Children[i].YMax = tree.YAxis
			}
		}
		if i&4 == 0 {
			tree.Children[i].ZMin = tree.ZAxis
			tree.Children[i].ZMax = tree.ZMax
		} else {
			tree.Children[i].ZMin = tree.ZMin
			tree.Children[i].ZMax = tree.ZAxis
		}
		err = constructOctree(&tree.Children[i], surfaces, tree.SurfaceIndices)
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
	root.ZMin = -1024
	root.ZMax = 1024

	err = constructOctree(root, surfaces, root.SurfaceIndices)
	if err != nil {
		fmt.Printf("Got error constructing octree!\n")
		root = nil
	}

	return

}

func (root *Octree) getTreeByPoint(point Point) (res *Octree) {

	current := root

	for {

		if point.X >= current.XMax || point.X < current.XMin {
			return
		}

		if point.Y >= current.YMax || point.Y < current.YMin {
			return
		}

		if point.Z >= current.ZMax || point.Z < current.ZMin {
			return
		}

		if current.Children == nil {
			break
		}
		if point.X >= current.XAxis {
			if point.Y < current.YAxis {
				if point.Z >= current.ZAxis {
					current = &current.Children[0]
					continue
				} else {
					current = &current.Children[4]
					continue
				}
			} else {
				if point.Z >= current.ZAxis {
					current = &current.Children[1]
					continue
				} else {
					current = &current.Children[5]
					continue
				}
			}
		} else {
			if point.Y >= current.YAxis {
				if point.Z >= current.ZAxis {
					current = &current.Children[2]
					continue
				} else {
					current = &current.Children[6]
					continue
				}
			} else {
				if point.Z >= current.ZAxis {
					current = &current.Children[3]
					continue
				} else {
					current = &current.Children[7]
					continue
				}
			}
		}
	}

	res = current
	return

}

func (tree *Octree) TraceVectorToColor(vec Vector, cache map[float64]int, cacheLock *sync.Mutex) (col Color) {
	surfaceIx, relativePoint, isResult := tree.TraceVector(vec, cache, cacheLock)
	if !isResult {
		col.R = 0
		col.G = 0
		col.B = 0
		return
	}
	col = tree.Surfaces[surfaceIx].GetColor(relativePoint)
	return
}

func (tree *Octree) TraceVector(vec Vector, cache map[float64]int, cacheLock *sync.Mutex) (resultIx int, relativePoint Point, isResult bool) {
	isResult = false
	resultIx = -1
	surfaces := tree.Surfaces
	closestDist2 := math.Pow(2048, 2)
	travelingVec := vec
	var closestRelativePoint, closestIntersection Point
	closestRelativePoint.X = math.NaN()
	closestRelativePoint.Y = math.NaN()
	closestRelativePoint.Z = math.NaN()
	startIx := 0
	if vec.Start.X <= vec.End.X {
		if vec.Start.Y > vec.End.Y {
			startIx = 1
		} else {
			startIx = 2
		}
	} else {
		if vec.Start.Y <= vec.End.Y {
			startIx = 3
		} else {
			startIx = 4
		}
	}
	if cache != nil {
		cacheLock.Lock()
		surfaceIx, isOk := cache[vec.End.X]
		cacheLock.Unlock()
		if isOk {
			closestRelativePoint = surfaces[surfaceIx].GetIntersection(vec)
		}
	}
	for {
		subtree := tree.getTreeByPoint(travelingVec.Start)
		if subtree == nil {
			return
		}
		for _, surfaceIx := range subtree.SurfaceIndices {
			intersection := surfaces[surfaceIx].GetIntersection(travelingVec)
			if !math.IsNaN(intersection.X) {
				currentDist2 := GetDist2(vec.Start, intersection)
				/*
					if cache != nil {
						if !cache.Contains(surfaceIx) {
							cache.Insort(surfaceIx, currentDist2)
						}
					}
				*/
				if currentDist2 <= closestDist2 {
					closestDist2 = currentDist2
					resultIx = surfaceIx
					closestIntersection = intersection
					//fmt.Printf("Setting closestIx: %d\n", resultIx)
				}
			}
		}
		// Because surfaces can span multiple trees, it's possible we have an endpoint
		// which is not the solution
		if !math.IsNaN(closestIntersection.X) && subtree.Id == tree.getTreeByPoint(closestIntersection).Id {
			closestRelativePoint = GetRelativePoint(vec.Start, closestIntersection)
			isResult = true
			if cache != nil {
				cacheLock.Lock()
				cache[vec.End.X] = resultIx
				cacheLock.Unlock()
			}
			return
		}
		neighbor := subtree.testBoundaries(travelingVec, startIx)
		if math.IsNaN(neighbor.X) {
			return
		}
		travelingVec.Start = neighbor
	}
}

func (tree *Octree) testBoundaries(vec Vector, startIx int) (res Point) {
	res.X = math.NaN()
	res.Y = math.NaN()
	res.Z = math.NaN()
	for i := 0; i < 6; i++ {
		var boundary Surface
		switch (startIx + i) % 6 {
		case 0:
			boundary = tree.ZMinBoundary()
		case 1:
			boundary = tree.YMinBoundary()
		case 2:
			boundary = tree.XMaxBoundary()
		case 3:
			boundary = tree.YMaxBoundary()
		case 4:
			boundary = tree.XMinBoundary()
		case 5:
			boundary = tree.ZMaxBoundary()
		default:
			return
		}
		res = boundary.GetIntersection(vec)
		if !math.IsNaN(res.X) {
			return
		}
	}
	return
}

func (tree *Octree) XMinBoundary() (res Surface) {
	res = &QuadSurface{
		V1: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMin - MILLI,
			},
		},
		V2: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMax + MILLI,
			},
		},
	}
	return
}

func (tree *Octree) XMaxBoundary() (res Surface) {
	res = &QuadSurface{
		V1: Vector{
			Start: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMin - MILLI,
			},
		},
		V2: Vector{
			Start: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMax + MILLI,
			},
		},
	}
	return
}

func (tree *Octree) YMinBoundary() (res Surface) {
	res = &QuadSurface{
		V1: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
		},
		V2: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMax + MILLI,
			},
		},
	}
	return
}

func (tree *Octree) YMaxBoundary() (res Surface) {
	res = &QuadSurface{
		V1: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMin - MILLI,
			},
		},
		V2: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMax + MILLI,
			},
		},
	}
	return
}

func (tree *Octree) ZMinBoundary() (res Surface) {
	res = &QuadSurface{
		V1: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
		},
		V2: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMin - MILLI,
			},
			End: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMin - MILLI,
			},
		},
	}
	return
}

func (tree *Octree) ZMaxBoundary() (res Surface) {
	res = &QuadSurface{
		V1: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMax + MILLI,
			},
			End: Point{
				X: tree.XMax + MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMax + MILLI,
			},
		},
		V2: Vector{
			Start: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMin - MILLI,
				Z: tree.ZMax + MILLI,
			},
			End: Point{
				X: tree.XMin - MILLI,
				Y: tree.YMax + MILLI,
				Z: tree.ZMax + MILLI,
			},
		},
	}
	return
}
