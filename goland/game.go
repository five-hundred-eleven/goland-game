package goland

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

const (
	End = iota
)

const (
	FRAMERATE          = 61
	NUMWORKERS         = 4 // TODO
	MILLI              = 1e-3
	MICRO              = 1e-6
	SHEARFACTOR        = 1.00
	WINDOWWIDTH  int32 = 1280
	WINDOWHEIGHT int32 = 960
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
	Players  []*Player `json:"players"`
	Surfaces []Surface `json:"surfaces"`
}

type Screen struct {
	Width, Height, Depth int32
	HAngles, VAngles     []float64
	Window               *sdl.Window
	Renderer             *sdl.Renderer
	SurfaceTexture       *sdl.Texture
	TargetTexture        *sdl.Texture
	SegTextures          []*sdl.Texture
	Segments             []*sdl.Rect
	TargetMask           *sdl.Rect
	SegMask              *sdl.Rect
	Format               *sdl.PixelFormat
}

type Player struct {
	Vector
	Velocity, RotVel float64
	Game             *Game
	Visited          map[int]bool
}

type Ray struct {
	PixelStart int32
	Data       []byte
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
		"Got player: %f %f %f\n",
		gameData.Players[0].X,
		gameData.Players[0].Y,
		gameData.Players[0].Theta,
	)
	fmt.Printf("Got num surfaces: %d\n", len(gameData.Surfaces))
	fmt.Printf(
		"Got surface: %f %f %f %f\n",
		gameData.Surfaces[0].P1.X,
		gameData.Surfaces[0].P1.Y,
		gameData.Surfaces[0].P2.X,
		gameData.Surfaces[0].P2.Y,
	)

	game = &Game{}
	game.Players = gameData.Players
	for _, player := range game.Players {
		player.Theta = player.Theta * math.Pi / 180.0
		player.Game = game
		player.Visited = make(map[int]bool)
	}
	game.Tree, err = NewOctreeFromSurfaces(gameData.Surfaces)

	return

}

func Advance(vec Vector, dist float64) (res Vector) {
	res = Vector{
		Point{
			vec.X + math.Sin(vec.Theta)*dist,
			vec.Y + math.Cos(vec.Theta)*dist,
			0.0, // TODO
		},
		vec.Theta,
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
	xp := p.Velocity * factor * math.Sin(p.Theta)
	pp := Point{
		X: p.X + xp*10,
		Y: p.Y,
	}
	intersection, _ := tree.TraceSegment(p.Point, pp)
	if math.IsNaN(intersection.X) {
		p.X += xp
	} else {
		p.X -= xp
	}
	yp := p.Velocity * factor * math.Cos(p.Theta)
	pp = Point{
		X: p.X,
		Y: p.Y + yp*10,
	}
	intersection, _ = tree.TraceSegment(p.Point, pp)
	if math.IsNaN(intersection.X) {
		p.Y += yp
	} else {
		p.Y -= yp
	}
	p.Theta = AddAndNormalize(p.Theta, p.RotVel)
	p.SetVisited(p.X, p.Y)
}

func (p *Player) rotate(rot float64) Vector {
	return Vector{Point{p.X, p.Y, 0.0}, AddAndNormalize(p.Theta, rot)}
}

func doRaytrace(completedCh chan int32, rayCh chan Ray, screen *Screen, segmentIx int32, game *Game, player *Player) {
	defer func() {
		completedCh <- segmentIx
	}()
	tree := game.Tree
	segment := screen.Segments[segmentIx]
	width := segment.W
	indexStart := segment.X
	indexStop := indexStart + width
	isfilled := make([]bool, width)
	dists := make([]float64, width)
	rotated := make([]Vector, width)
	for i := indexStart; i < indexStop; i++ {
		col := i - indexStart
		rotated[col] = player.rotate(screen.HAngles[i])
		endpoint, dist := tree.TraceVector(rotated[col], 4096.0)
		if math.IsNaN(endpoint.X) {
			continue
		}
		isfilled[col] = true
		dists[col] = dist
	}
	for row := int32(0); row < screen.Height; row++ {
		rowStart := (row*screen.Width + indexStart) * 4
		data := make([]byte, width*4)
		isUpper := math.Signbit(screen.VAngles[row])
		abssin := math.Abs(math.Sin(screen.VAngles[row]))
		floorDist := math.NaN()
		ceilDist := math.NaN()
		if !isUpper {
			floorDist = math.Abs(FLOORHEIGHT / math.Tan(screen.VAngles[row]))
		} else {
			ceilDist = math.Abs(CEILHEIGHT / math.Tan(screen.VAngles[row]))
		}
		for col := int32(0); col < width; col++ {
			colStart := col * 4
			if isfilled[col] {
				// TODO don't do this repeatedly
				// should probably refactor this whole function...
				xDist := dists[col]
				yDist := abssin * xDist
				if yDist < WALLHEIGHT {
					x := byte(255 - math.Min(math.Log(xDist+yDist*2)*45, 245.0))
					data[colStart] = x
					data[colStart+1] = x
					data[colStart+2] = x
					continue
				}
			}
			if isUpper {
				if ceilDist < 256 {
					rayvec := rotated[col]
					ceilpoint1 := Advance(rayvec, ceilDist)
					if player.GetVisited(ceilpoint1.X, ceilpoint1.Y) {
						data[colStart] = 0
						data[colStart+1] = 0
						data[colStart+2] = 255
						continue
					}
				}
			} else {
				if floorDist < 256 {
					x := byte(34 - math.Log(floorDist)*6)
					data[colStart] = x
					data[colStart+1] = x
					data[colStart+2] = x
					continue
				}
			}
		}
		rayCh <- Ray{rowStart, data}
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
				go doRaytrace(completedCh, rayCh, screen, i, game, player)
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
