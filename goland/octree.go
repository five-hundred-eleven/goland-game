package goland

import (
	"fmt"
	"math"
	"sync"
)

var NaN = math.NaN()

type Octree struct {
	Id                   int
	SurfaceIndices       []int
	Surfaces             []QuadSurface
	Children             []Octree
	Parent               *Octree
	XMinF, XMaxF, XAxisF float64
	YMinF, YMaxF, YAxisF float64
	ZMinF, ZMaxF, ZAxisF float64
	XMin, XMax, XAxis    int
	YMin, YMax, YAxis    int
	ZMin, ZMax, ZAxis    int
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

func (tree *Octree) getSurfacesInTree(surfaces []QuadSurface, parentSurfaceIndices []int) (res []int) {
	res = make([]int, 0)
	for _, ix := range parentSurfaceIndices {
		surface := surfaces[ix]
		points := surface.GetPoints()
		doContinue := false
		for _, point := range points {
			if tree.XMinF <= point.X && point.X <= tree.XMaxF && tree.YMinF <= point.Y && point.Y <= tree.YMaxF && tree.ZMinF <= point.Z && point.Z <= tree.ZMaxF {
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
			neighbor := tree.testBoundaries(vec, []int{0, 1, 2, 3, 4, 5})
			if !math.IsNaN(neighbor.X) {
				//fmt.Printf("Storing surface in tree by intersection: %d\n", tree.Id)
				res = append(res, ix)
				break
			}
		}
	}
	return
}

func constructOctree(tree *Octree, surfaces []QuadSurface, parentSurfaceIndices []int) (err error) {

	tree.Id = TREEID
	TREEID++

	tree.XAxis = (tree.XMin + tree.XMax) / 2
	tree.YAxis = (tree.YMin + tree.YMax) / 2
	tree.ZAxis = (tree.ZMin + tree.ZMax) / 2

	tree.XMinF = float64(tree.XMin)
	tree.XMaxF = float64(tree.XMax)
	tree.XAxisF = float64(tree.XAxis)
	tree.YMinF = float64(tree.YMin)
	tree.YMaxF = float64(tree.YMax)
	tree.YAxisF = float64(tree.YAxis)
	tree.ZMinF = float64(tree.ZMin)
	tree.ZMaxF = float64(tree.ZMax)
	tree.ZAxisF = float64(tree.ZAxis)

	tree.SurfaceIndices = tree.getSurfacesInTree(surfaces, parentSurfaceIndices)

	// no need to recurse if small number of surfaces
	if len(tree.SurfaceIndices) <= MINSURFACES {
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

func NewOctreeFromSurfaces(surfaces []QuadSurface) (root *Octree, err error) {

	root = &Octree{}
	root.SurfaceIndices = make([]int, len(surfaces))
	for i := 0; i < len(surfaces); i++ {
		root.SurfaceIndices[i] = i
	}
	// TODO figure out min/max dynamically??
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

func (root *Octree) GetTreeByPointF(point Point) (res *Octree) {

	current := root
	X := point.X
	Y := point.Y
	Z := point.Z

	for {

		// TODO remove me
		/*
			if X >= current.XMaxF || X < current.XMinF {
				return
			}

			if Y >= current.YMaxF || Y < current.YMinF {
				return
			}

			if Z >= current.ZMaxF || Z < current.ZMinF {
				return
			}
		*/

		if current.Children == nil {
			break
		}
		if X >= current.XAxisF {
			if Y < current.YAxisF {
				if Z >= current.ZAxisF {
					current = &current.Children[0]
					continue
				} else {
					current = &current.Children[4]
					continue
				}
			} else {
				if Z >= current.ZAxisF {
					current = &current.Children[1]
					continue
				} else {
					current = &current.Children[5]
					continue
				}
			}
		} else {
			if Y >= current.YAxisF {
				if Z >= current.ZAxisF {
					current = &current.Children[2]
					continue
				} else {
					current = &current.Children[6]
					continue
				}
			} else {
				if Z >= current.ZAxisF {
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

func (root *Octree) GetTreeByPoint(point Point) (res *Octree) {

	current := root
	X := int(math.Floor(point.X))
	Y := int(math.Floor(point.Y))
	Z := int(math.Floor(point.Z))

	for {

		// TODO remove me
		/*
			if X >= current.XMaxF || X < current.XMinF {
				return
			}

			if Y >= current.YMaxF || Y < current.YMinF {
				return
			}

			if Z >= current.ZMaxF || Z < current.ZMinF {
				return
			}
		*/

		if current.Children == nil {
			break
		}
		if X >= current.XAxis {
			if Y < current.YAxis {
				if Z >= current.ZAxis {
					current = &current.Children[0]
					continue
				} else {
					current = &current.Children[4]
					continue
				}
			} else {
				if Z >= current.ZAxis {
					current = &current.Children[1]
					continue
				} else {
					current = &current.Children[5]
					continue
				}
			}
		} else {
			if Y >= current.YAxis {
				if Z >= current.ZAxis {
					current = &current.Children[2]
					continue
				} else {
					current = &current.Children[6]
					continue
				}
			} else {
				if Z >= current.ZAxis {
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

func (tree *Octree) TraceVectorToColor(vec Vector, cache map[float64]int, cacheLock *sync.Mutex, boundariesOrder []int) (col Color) {
	result := tree.TraceVector(vec, cache, cacheLock, boundariesOrder)
	if result == nil {
		col.R = 0
		col.G = 0
		col.B = 0
		return
	}
	col = tree.Surfaces[result.SurfaceId].GetColor(result)
	return
}

func (tree *Octree) TraceVector(vec Vector, cache map[float64]int, cacheLock *sync.Mutex, boundariesOrder []int) (res *RayResult) {
	surfaces := tree.Surfaces
	travelingVec := vec
	visited := make(map[int]bool)
	prevTreeId := tree.Id
	/*
		if cache != nil {
			cacheLock.Lock()
			surfaceId, isOk := cache[vec.End.X]
			cacheLock.Unlock()
			if isOk {
				closestIntersection = surfaces[surfaceId].GetIntersection(vec)
				relativePoint = GetRelativePoint(vec.Start, closestIntersection)
				//closestDist2 = GetDist2(vec.Start, closestIntersection)
				resultIx = surfaceId
				return
				//visited[surfaceId] = true
			}
		}
	*/
	iter := 0
	for {
		iter++
		subtree := tree.GetTreeByPoint(travelingVec.Start)
		if subtree == nil {
			return
		}
		if subtree.Id == prevTreeId {
			//fmt.Printf("Got subtree with identical id: (%f, %f, %f) -> (%f, %f, %f)\n", vec.Start.X, vec.Start.Y, vec.Start.Z, vec.End.X, vec.End.Y, vec.End.Z)
			return
		}
		prevTreeId = subtree.Id
		/*
			subtreeF := tree.GetTreeByPointF(travelingVec.Start)
			if subtree.Id != subtreeF.Id {
				fmt.Printf("Tree discrepency: (%f, %f, %f)\n", travelingVec.Start.X, travelingVec.Start.Y, travelingVec.Start.Z)
			}
		*/
		for _, surfaceId := range subtree.SurfaceIndices {
			_, isOk := visited[surfaceId]
			if isOk {
				continue
			}
			visited[surfaceId] = true
			intersection := surfaces[surfaceId].GetIntersection(travelingVec)
			if !math.IsNaN(intersection.X) {
				currentDist2 := GetDist2(vec.Start, intersection)
				/*
					if cache != nil {
						if !cache.Contains(surfaceId) {
							cache.Insort(surfaceId, currentDist2)
						}
					}
				*/
				if res == nil {
					res = &RayResult{
						Intersection: intersection,
						Dist2:        currentDist2,
						SurfaceId:    surfaceId,
						IsFinal:      true,
					}
				} else if currentDist2 < res.Dist2 {
					res.Intersection = intersection
					res.Dist2 = currentDist2
					res.SurfaceId = surfaceId
				}
			}
		}
		// Because surfaces can span multiple trees, it's possible we have an endpoint
		// which is not the solution
		if res != nil && subtree.Id == tree.GetTreeByPoint(res.Intersection).Id {
			/*
				if cache != nil {
					cacheLock.Lock()
					cache[vec.End.X] = resultIx
					cacheLock.Unlock()
				}
			*/
			//fmt.Printf("Got result after %d iterations\n", iter)
			return
		}
		neighbor := subtree.testBoundaries(travelingVec, boundariesOrder)
		if math.IsNaN(neighbor.X) {
			return
		}
		travelingVec.Start = neighbor
	}
}

func (tree *Octree) testBoundaries(vec Vector, ordering []int) (res Point) {
	res.X = NaN
	res.Y = NaN
	res.Z = NaN
	for i := range ordering {
		var boundary *QuadSurface
		switch i {
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
			//fmt.Printf("Got result: at i=%d, startIx=%d\n", i, startIx)
			return
		}
	}
	return
}

func (tree *Octree) XMinBoundary() (res *QuadSurface) {
	res = &QuadSurface{
		P1: Point{
			X: tree.XMinF - MILLI,
			Y: tree.YMinF,
			Z: tree.ZMinF,
		},
		P2: Point{
			X: tree.XMinF - MILLI,
			Y: tree.YMaxF,
			Z: tree.ZMinF,
		},
		P3: Point{
			X: tree.XMinF - MILLI,
			Y: tree.YMinF,
			Z: tree.ZMaxF,
		},
	}
	return
}

func (tree *Octree) XMaxBoundary() (res *QuadSurface) {
	res = &QuadSurface{
		P1: Point{
			X: tree.XMaxF + MILLI,
			Y: tree.YMinF,
			Z: tree.ZMinF,
		},
		P2: Point{
			X: tree.XMaxF + MILLI,
			Y: tree.YMaxF,
			Z: tree.ZMinF,
		},
		P3: Point{
			X: tree.XMaxF + MILLI,
			Y: tree.YMinF,
			Z: tree.ZMaxF,
		},
	}
	return
}

func (tree *Octree) YMinBoundary() (res *QuadSurface) {
	res = &QuadSurface{
		P1: Point{
			X: tree.XMinF,
			Y: tree.YMinF - MILLI,
			Z: tree.ZMinF,
		},
		P2: Point{
			X: tree.XMaxF,
			Y: tree.YMinF - MILLI,
			Z: tree.ZMinF,
		},
		P3: Point{
			X: tree.XMinF,
			Y: tree.YMinF - MILLI,
			Z: tree.ZMaxF,
		},
	}
	return
}

func (tree *Octree) YMaxBoundary() (res *QuadSurface) {
	res = &QuadSurface{
		P1: Point{
			X: tree.XMinF,
			Y: tree.YMaxF + MILLI,
			Z: tree.ZMinF,
		},
		P2: Point{
			X: tree.XMaxF,
			Y: tree.YMaxF + MILLI,
			Z: tree.ZMinF,
		},
		P3: Point{
			X: tree.XMinF,
			Y: tree.YMaxF + MILLI,
			Z: tree.ZMaxF,
		},
	}
	return
}

func (tree *Octree) ZMinBoundary() (res *QuadSurface) {
	res = &QuadSurface{
		P1: Point{
			X: tree.XMinF,
			Y: tree.YMinF,
			Z: tree.ZMinF - MILLI,
		},
		P2: Point{
			X: tree.XMaxF,
			Y: tree.YMinF,
			Z: tree.ZMinF - MILLI,
		},
		P3: Point{
			X: tree.XMinF,
			Y: tree.YMaxF,
			Z: tree.ZMinF - MILLI,
		},
	}
	return
}

func (tree *Octree) ZMaxBoundary() (res *QuadSurface) {
	res = &QuadSurface{
		P1: Point{
			X: tree.XMinF,
			Y: tree.YMinF,
			Z: tree.ZMaxF + MILLI,
		},
		P2: Point{
			X: tree.XMaxF,
			Y: tree.YMinF,
			Z: tree.ZMaxF + MILLI,
		},
		P3: Point{
			X: tree.XMinF,
			Y: tree.YMaxF,
			Z: tree.ZMaxF + MILLI,
		},
	}
	return
}
