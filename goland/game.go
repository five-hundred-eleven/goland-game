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
	NUMWORKERS         = 8 // TODO
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
	surfaces := make([]Surface, len(gameData.Surfaces))
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
	_, _, isResult := tree.TraceVector(vec, nil)
	if !isResult {
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
	_, _, isResult = tree.TraceVector(vec, nil)
	if !isResult {
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

func doRaytrace(completedCh chan int32, rayCh chan Ray, screen *Screen, segmentIx int32, game *Game, player *Player, cache *SurfaceCache) {
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
	XOffset := screen.XOffsets[segmentIx]
	HAngles := screen.HAngles
	VAngles := screen.VAngles
	for y := YStart; y < YStop; y++ {
		rowStart := y * screen.Width * 4
		data := make([]byte, width*4)
		/*
			isUpper := math.Signbit(screen.VAngles[y])
				abssin := math.Abs(math.Sin(screen.VAngles[y]))
				floorDist := math.NaN()
				ceilDist := math.NaN()
				if !isUpper {
					floorDist = math.Abs(FLOORHEIGHT / math.Tan(screen.VAngles[y]))
				} else {
					ceilDist = math.Abs(CEILHEIGHT / math.Tan(screen.VAngles[y]))
				}
		*/
		for xi := XStart; xi < XStop; xi++ {
			x := (xi + XOffset) % screen.Width
			pixelStart := x * 4
			vec := player.VectorFromThetas(HAngles[x], VAngles[y])
			var color Color
			color = tree.TraceVectorToColor(vec, cache)
			data[pixelStart] = color.B
			data[pixelStart+1] = color.G
			data[pixelStart+2] = color.R
			/*
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
						logdist := math.Log(floorDist)
						r := byte(34 - logdist*7)
						g := byte(34 - logdist*6)
						b := byte(34 - logdist*5)
						data[colStart] = b
						data[colStart+1] = g
						data[colStart+2] = r
						continue
					}
				}
			*/
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
			cache := NewSurfaceCache()
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
				go doRaytrace(completedCh, rayCh, screen, i, game, player, cache)
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
