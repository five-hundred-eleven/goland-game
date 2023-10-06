package goland

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	End = iota
)

const (
	FRAMERATE          = 61
	NUMWORKERS         = 8 // TODO
	MILLI              = 1e-3
	MICRO              = 1e-6
	SHEARFACTOR        = 1.00
	WINDOWWIDTH  int32 = 320
	WINDOWHEIGHT int32 = 240
	VELOCITY           = 0.25
	ROTVEL             = 0.075
	WALLHEIGHT         = 2.5
	FLOORHEIGHT        = -2.5
	CEILHEIGHT         = 3.5
)

var RANDOMSAMPLE = make([]byte, 4096)

type Game struct {
	Players []*Player
	Tree    *Octree
}

type GameData struct {
	Players  []*Player     `json:"players"`
	Surfaces []SurfaceData `json:"surfaces"`
}

type SurfaceData []Point

type Screen struct {
	Width, Height, Depth int32
	HAngles, VAngles     []float64
	Window               *sdl.Window
	Renderer             *sdl.Renderer
	SurfaceTexture       *sdl.Texture
	TargetTexture        *sdl.Texture
	SegTextures          []*sdl.Texture
	Segments             []*sdl.Rect
	XOffsets             []int32
	TargetMask           *sdl.Rect
	SegMask              *sdl.Rect
	Format               *sdl.PixelFormat
}

type Player struct {
	X                float64 `json:"x"`
	Y                float64 `json:"y"`
	Z                float64 `json:"z"`
	Height           float64 `json:"height"`
	HTheta           float64 `json:"htheta"`
	VTheta           float64 `json:"vtheta"`
	Velocity, RotVel float64
	Game             *Game
	Visited          map[int]bool
}

type Ray struct {
	PixelStart int32
	Data       []byte
}

type FrontierPoint struct {
	SurfaceId int
	// TODO is this needed?
	PixelId int32
	X, Y    int32
}

type RenderingContext struct {
	Xs, Ys, Zs             []float64
	Width, Height          int32
	tree                   *Octree
	VisitedSurfaces        map[int][]int32
	RayResults             []*RayResult
	Frontier, NextFrontier []*FrontierPoint
}

func NewGameFromFilename(filename string) (game *Game, err error) {

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("error opening maze file: %s\n", err)
		return
	}

	gameData := GameData{}
	err = json.Unmarshal(content, &gameData)
	if err != nil {
		fmt.Printf("error unmarshaling maze file: %s\n", err)
		return
	}

	fmt.Printf("Got game with num players: %d\n", len(gameData.Players))
	fmt.Printf(
		"Got player: %f %f %f %f\n",
		gameData.Players[0].X,
		gameData.Players[0].Y,
		gameData.Players[0].HTheta,
		gameData.Players[0].VTheta,
	)

	game = &Game{}
	game.Players = gameData.Players
	for _, player := range game.Players {
		player.Z += player.Height
		player.HTheta = player.HTheta * math.Pi / 180.0
		player.VTheta = player.VTheta * math.Pi / 180.0
		player.Game = game
		player.Visited = make(map[int]bool)
	}
	surfaces := make([]QuadSurface, len(gameData.Surfaces))
	for i, surface := range gameData.Surfaces {
		surfaces[i], err = SurfaceFromSurfaceData(surface)
		if err != nil {
			fmt.Printf("Got err constructing surface: %s\n", err)
			return
		}
		surfaces[i].SetColor(Color{R: 255, G: 255, B: 255})
	}
	fmt.Printf("Got num surfaces: %d\n", len(surfaces))
	game.Tree, err = NewOctreeFromSurfaces(surfaces)
	if err != nil {
		game = nil
		return
	}

	return

}

func visitedIndex(X, Y float64) int {
	return (int(X)+2048)*4096 + int(Y)
}

func (p *Player) SetVisited(X, Y float64) {
	p.Visited[visitedIndex(X, Y)] = true
}

func (p *Player) GetVisited(X, Y float64) (res bool) {
	res, ok := p.Visited[visitedIndex(X, Y)]
	if !ok {
		res = false
	}
	return
}

func (p *Player) Move(factor float64) {
	tree := p.Game.Tree
	xp := p.Velocity * factor * math.Sin(p.HTheta)
	vec := Vector{
		Start: Point{
			X: p.X,
			Y: p.Y,
			Z: p.Z,
		},
		End: Point{
			X: p.X + xp,
			Y: p.Y,
			Z: p.Z,
		},
	}
	result := tree.TraceVector(vec, nil, nil, []int{0, 1, 2, 3, 4, 5})
	if result == nil {
		p.X += xp
	} else {
		p.X -= xp
	}
	yp := p.Velocity * factor * math.Cos(p.HTheta)
	vec = Vector{
		Start: Point{
			X: p.X,
			Y: p.Y,
			Z: p.Z,
		},
		End: Point{
			X: p.X,
			Y: p.Y + yp,
			Z: p.Z,
		},
	}
	result = tree.TraceVector(vec, nil, nil, []int{0, 1, 2, 3, 4, 5})
	if result == nil {
		p.Y += yp
	} else {
		p.Y -= yp
	}
	p.HTheta = AddAndNormalize(p.HTheta, p.RotVel)
	p.SetVisited(p.X, p.Y)
}

func (p *Player) VectorFromThetas(HTheta, VTheta float64) (res Vector) {

	res.Start.X = p.X
	res.Start.Y = p.Y
	res.Start.Z = p.Z

	HTheta = AddAndNormalize(p.HTheta, HTheta)
	VTheta = AddAndNormalize(p.VTheta, VTheta)

	res.End.X = res.Start.X + math.Sin(HTheta)*4096.0
	res.End.Y = res.Start.Y + math.Cos(HTheta)*4096.0
	res.End.Z = res.Start.Z + math.Cos(VTheta)*4096.0

	return

}

func ExplorePixel(renderingCtx *RenderingContext, vec Vector, pixelId int32) (rayResult *RayResult) {

	boundariesOrdering := []int{0, 0, 0, 0, 0, 0}
	if vec.Start.X < vec.End.X {
		boundariesOrdering[0] = 2
		boundariesOrdering[3] = 4
	} else {
		boundariesOrdering[0] = 4
		boundariesOrdering[3] = 2
	}
	if vec.Start.Y < vec.End.Y {
		boundariesOrdering[1] = 3
		boundariesOrdering[4] = 1
	} else {
		boundariesOrdering[1] = 1
		boundariesOrdering[4] = 3
	}
	if vec.Start.Z < vec.End.Z {
		boundariesOrdering[2] = 5
		boundariesOrdering[5] = 0
	} else {
		boundariesOrdering[2] = 0
		boundariesOrdering[5] = 5
	}

	rayResult = renderingCtx.tree.TraceVector(vec, nil, nil, boundariesOrdering)
	if rayResult == nil {
		return
	}

	pixelIds, isOk := renderingCtx.VisitedSurfaces[rayResult.SurfaceId]
	if !isOk {
		pixelIds = make([]int32, 1, 4)
		pixelIds[0] = pixelId
	} else {
		pixelIds = append(pixelIds, pixelId)
	}
	renderingCtx.VisitedSurfaces[rayResult.SurfaceId] = pixelIds
	renderingCtx.RayResults[rayResult.SurfaceId] = rayResult
	return

}

func VisitIfUnvisited(renderingCtx *RenderingContext, fp *FrontierPoint) {
	rayResult := renderingCtx.RayResults[fp.PixelId]
	if rayResult != nil && rayResult.IsFinal {
		return
	}
	pixelIds, isOk := renderingCtx.VisitedSurfaces[fp.SurfaceId]
	if !isOk {
		pixelIds = make([]int32, 1, 4)
		pixelIds[0] = fp.PixelId
	} else {
		for _, visitedPixelId := range pixelIds {
			if fp.PixelId == visitedPixelId {
				return
			}
		}
		pixelIds = append(pixelIds, fp.PixelId)
	}
	renderingCtx.VisitedSurfaces[fp.SurfaceId] = pixelIds
	renderingCtx.NextFrontier = append(renderingCtx.NextFrontier, fp)
	return
}

func ExpandPixel(renderingCtx *RenderingContext, fp *FrontierPoint) {

	w := renderingCtx.Width
	h := renderingCtx.Height
	surfaceId := fp.SurfaceId
	y := fp.Y
	x := fp.X

	if x > 0 {
		fp := &FrontierPoint{
			SurfaceId: surfaceId,
			PixelId:   y*w + x - 1,
			X:         x - 1,
			Y:         y,
		}
		VisitIfUnvisited(renderingCtx, fp)
	}
	if x+1 < w {
		fp := &FrontierPoint{
			SurfaceId: surfaceId,
			PixelId:   y*w + x + 1,
			X:         x + 1,
			Y:         y,
		}
		VisitIfUnvisited(renderingCtx, fp)
	}
	if y > 0 {
		fp := &FrontierPoint{
			SurfaceId: surfaceId,
			PixelId:   (y-1)*w + x,
			X:         x,
			Y:         y - 1,
		}
		VisitIfUnvisited(renderingCtx, fp)
	}
	if y+1 < h {
		fp := &FrontierPoint{
			SurfaceId: surfaceId,
			PixelId:   (y+1)*w + x,
			X:         x,
			Y:         y + 1,
		}
		VisitIfUnvisited(renderingCtx, fp)
	}

	return

}

func DoRender(completedCh chan int32, rayCh chan Ray, screen *Screen, segmentIx int32, game *Game, player *Player, cache map[float64]int, cacheLock *sync.Mutex) {
	defer func() {
		completedCh <- segmentIx
	}()
	tree := game.Tree
	segment := screen.Segments[segmentIx]
	width := segment.W
	height := segment.H
	XStart := segment.X
	YStart := segment.Y
	XStop := XStart + width
	YStop := YStart + height
	//XOffset := screen.XOffsets[segmentIx]
	HAngles := screen.HAngles
	VAngles := screen.VAngles

	vec := Vector{}
	vec.Start.X = player.X
	vec.Start.Y = player.Y
	vec.Start.Z = player.Z

	xs := make([]float64, width)
	ys := make([]float64, width)
	zs := make([]float64, height)

	// visitedSurfaces is a map of surface IDs to pixels that have visited them
	visitedSurfaces := make(map[int][]int32)
	rayResults := make([]*RayResult, height*width)
	nextFrontier := make([]*FrontierPoint, 0, 1024)

	renderingCtx := &RenderingContext{}
	renderingCtx.Xs = xs
	renderingCtx.Ys = ys
	renderingCtx.Zs = zs
	renderingCtx.Width = width
	renderingCtx.Height = height
	renderingCtx.tree = tree
	renderingCtx.VisitedSurfaces = visitedSurfaces
	renderingCtx.RayResults = rayResults
	renderingCtx.NextFrontier = nextFrontier

	//pixels := make([]byte, height*width*4)

	for xi := XStart; xi < XStop; xi++ {
		HTheta := AddAndNormalize(player.HTheta, HAngles[xi])
		x := xi - XStart
		xs[x] = vec.Start.X + math.Sin(HTheta)*4096.0
		ys[x] = vec.Start.Y + math.Cos(HTheta)*4096.0
	}
	for yi := YStart; yi < YStop; yi++ {
		VTheta := AddAndNormalize(player.VTheta, VAngles[yi])
		y := yi - YStart
		zs[y] = player.Z + math.Cos(VTheta)*4096.0
	}
	for y := int32(4); y < height; y += 8 {
		vec.End.Z = zs[y]
		for x := int32(4); x < width; x += 8 {
			pixelId := y*width + x
			vec.End.X = xs[x]
			vec.End.Y = ys[x]
			rayResult := ExplorePixel(renderingCtx, vec, pixelId)
			if rayResult == nil {
				continue
			}
			fp := &FrontierPoint{
				SurfaceId: rayResult.SurfaceId,
				PixelId:   pixelId,
				X:         x,
				Y:         y,
			}
			ExpandPixel(renderingCtx, fp)
		}
	}
	for {
		if len(renderingCtx.NextFrontier) == 0 {
			break
		}
		renderingCtx.Frontier = renderingCtx.NextFrontier
		renderingCtx.NextFrontier = make([]*FrontierPoint, 0, len(renderingCtx.Frontier)*4)
		for _, fp := range renderingCtx.Frontier {
			surface := tree.Surfaces[fp.SurfaceId]
			vec.End.X = xs[fp.X]
			vec.End.Y = ys[fp.X]
			vec.End.Z = zs[fp.Y]
			var rayResult *RayResult
			intersection := surface.GetIntersection(vec)
			if math.IsNaN(intersection.X) {
				continue
				/*
					rayResult = ExplorePixel(renderingCtx, vec, fp.PixelId)
					if rayResult == nil {
						continue
					}
				*/
			}
			dist2 := GetDist2(vec.Start, intersection)
			if rayResults[fp.PixelId] != nil && dist2 > rayResults[fp.PixelId].Dist2 {
				continue
			}
			rayResult = &RayResult{
				Intersection: intersection,
				Dist2:        dist2,
				SurfaceId:    fp.SurfaceId,
				IsFinal:      false,
			}
			rayResults[fp.PixelId] = rayResult
			fp = &FrontierPoint{
				SurfaceId: fp.SurfaceId,
				PixelId:   fp.PixelId,
				X:         fp.X,
				Y:         fp.Y,
			}
			ExpandPixel(renderingCtx, fp)
		}
	}
	for yi := YStart; yi < YStop; yi++ {
		y := yi - YStart
		data := make([]byte, width*4)
		for x := int32(0); x < width; x++ {
			pixelId := y*width + x
			rayResult := rayResults[pixelId]
			if rayResult == nil {
				data[x*4+2] = 0
				data[x*4+1] = 0
				data[x*4] = 0
				continue
			}
			surface := tree.Surfaces[rayResult.SurfaceId]
			c := surface.GetColor(rayResult)
			data[x*4+2] = c.R
			data[x*4+1] = c.B
			data[x*4] = c.G
		}
		rayCh <- Ray{yi * width * 4, data}
	}
}

func DoMaze(recvCh chan int, sendCh chan int, screen *Screen, game *Game) {
	defer func() {
		sendCh <- End
	}()

	fmt.Printf("In DoMaze()\n")

	FRAMESLEEP := int64(math.Floor(1.0 / FRAMERATE * 1e9))
	FRAMESLEEPF := float64(FRAMESLEEP)
	renderer := screen.Renderer
	targetTexture := screen.TargetTexture

	screen.HAngles = make([]float64, screen.Width)
	widthF := float64(screen.Width)
	depthF := float64(screen.Depth)
	r := widthF/2 + 0.5
	rRaised := math.Pow(r, SHEARFACTOR)
	exp := math.Log(r) / math.Log(rRaised)
	inc := rRaised / widthF
	L := (screen.Width - 1) / 2
	R := screen.Width / 2
	for i := int32(0); i < screen.Width/2; i++ {
		xBase := 0.5 + (float64(i) * inc)
		xRaised := math.Pow(xBase, exp)
		xFinal := math.Atan(xRaised / depthF)
		il := L - i
		ir := R + i
		screen.HAngles[il] = -xFinal
		screen.HAngles[ir] = xFinal
	}

	screen.VAngles = make([]float64, screen.Height)
	heightF := float64(screen.Height)
	r = heightF/2 + 0.5
	rRaised = math.Pow(r, SHEARFACTOR)
	exp = math.Log(r) / math.Log(rRaised)
	inc = rRaised / widthF
	U := (screen.Height - 1) / 2
	D := screen.Height / 2
	for i := int32(0); i < screen.Height/2; i++ {
		xBase := 0.5 + (float64(i) * inc)
		xRaised := math.Pow(xBase, exp)
		xFinal := math.Atan(xRaised / depthF)
		iu := U - i
		id := D + i
		screen.VAngles[iu] = -xFinal
		screen.VAngles[id] = xFinal
	}

	completedCh := make(chan int32)
	rayCh := make(chan Ray)
	intervalDurSec := int64(1e9)
	intervalStartSec := time.Now().UnixNano()
	numFrames := 0
	player := game.Players[0]
	numCompleted := 0
	for {
		startTime := time.Now().UnixNano()
		if startTime-intervalStartSec >= intervalDurSec {
			fmt.Printf("FPS: %d\n", numFrames)
			fmt.Printf("Player: (%f, %f) facing (%f, %f)\n", player.X, player.Y, player.HTheta, player.VTheta)
			intervalStartSec = startTime
			numFrames = 0
		}
		select {
		case msg := <-recvCh:
			if msg == End {
				return
			}
			fmt.Printf("you goofed\n")
		default:
			var err error
			cache := make(map[float64]int)
			cacheLock := &sync.Mutex{}
			err = renderer.SetDrawColor(0, 0, 0, 255)
			if err != nil {
				fmt.Printf("Got error on SetDrawColor(): %s\n", err)
				return
			}
			err = renderer.Clear()
			if err != nil {
				fmt.Printf("Got error on Clear(): %s\n", err)
				return
			}
			numCompleted = 0
			if len(completedCh) != 0 {
				fmt.Printf("completedCh should be empty\n")
				return
			}
			for i := int32(0); i < NUMWORKERS; i++ {
				go DoRender(completedCh, rayCh, screen, i, game, player, cache, cacheLock)
			}
			pixels, _, err := targetTexture.Lock(nil)
			if err != nil {
				fmt.Printf("Got err from Lock(): %s\n", err)
				return
			}
			for {
				select {
				case <-completedCh:
					numCompleted++
				case rayItem := <-rayCh:
					pixelStart := rayItem.PixelStart
					pixelStop := pixelStart + int32(len(rayItem.Data))
					copy(pixels[pixelStart:pixelStop], rayItem.Data)
				}
				if numCompleted >= NUMWORKERS {
					break
				}
			}
			// println("got past loop")
			targetTexture.Unlock()
			renderer.Copy(targetTexture, nil, nil)
			renderer.Present()
		}
		dur := time.Now().UnixNano() - startTime
		factor := 1.0
		if FRAMESLEEP >= dur {
			time.Sleep(time.Duration(FRAMESLEEP-dur) * time.Nanosecond)
		} else {
			factor += (float64(dur) - FRAMESLEEPF) / FRAMESLEEPF
		}
		numFrames++
		game.Players[0].Move(factor)
	}
}
