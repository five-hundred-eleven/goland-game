package goland

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	End = iota
)

const (
	FRAMERATE          = 61
	NUMWORKERS         = 4 // TODO
	MIDNIGHT           = 0.0
	ONETHIRTY          = math.Pi * 0.25
	THREEO             = math.Pi * 0.5
	FOURTHIRTY         = math.Pi * 0.75
	SIXO               = math.Pi * 1.0
	SEVENTHIRTY        = math.Pi * 1.25
	NINEO              = math.Pi * 1.5
	TENTHIRTY          = math.Pi * 1.75
	TWOPI              = math.Pi * 2.0
	MICRO              = 1e-6
	SHEARFACTOR        = 1.00
	WINDOWWIDTH  int32 = 1280
	WINDOWHEIGHT int32 = 960
	VELOCITY           = 0.5
	ROTVEL             = 0.075
	WALLHEIGHT         = 5.0
	FLOORHEIGHT        = -5.0
	CEILHEIGHT         = 6.5
)

type Point struct {
	X, Y float64
}

type Vector struct {
	Point
	Dir float64
}

type Game struct {
	Cols, Rows int
	Grid       []bool
	Players    []*Player
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
	Visited          []bool
}

type Ray struct {
	PixelStart int32
	Data       []byte
}

func NewGameFromFilename(filename string) (game *Game, err error) {

	f, err := os.Open(filename)
	if err != nil {
		println("error opening maze file")
		return
	}
	numBytesRead := 0
	scanner := bufio.NewScanner(f)
	scanner.Scan()
	startYStr := scanner.Text()
	numBytesRead += len(startYStr) + 1
	scanner.Scan()
	startXStr := scanner.Text()
	numBytesRead += len(startXStr) + 1
	scanner.Scan()
	startDirectionStr := scanner.Text()
	numBytesRead += len(startDirectionStr) + 1
	scanner.Scan()
	numRowsStr := scanner.Text()
	numBytesRead += len(numRowsStr) + 1
	scanner.Scan()
	numColsStr := scanner.Text()
	numBytesRead += len(numColsStr) + 1

	startY, err := strconv.ParseFloat(startYStr, 32)
	if err != nil {
		println(fmt.Sprintf("error parsing starting Y coordinate: %s", err))
		return
	}

	startX, err := strconv.ParseFloat(startXStr, 32)
	if err != nil {
		println(fmt.Sprintf("error parsing starting X coordinate: %s", err))
		return
	}

	startDirection, err := strconv.ParseFloat(startDirectionStr, 32)
	if err != nil {
		println(fmt.Sprintf("error parsing starting direction: %s", err))
		return
	}
	// convert to radians
	startDirection = startDirection * math.Pi / 180.0

	println(fmt.Sprintf("Got starting coordinates: (%f, %f) facing %f", startX, startY, startDirection))

	numRows, err := strconv.Atoi(numRowsStr)
	if err != nil {
		println(fmt.Sprintf("error parsing number of rows: %s", err))
		return
	}

	numCols, err := strconv.Atoi(numColsStr)
	if err != nil {
		println(fmt.Sprintf("error parsing number of cols: %s", err))
		return
	}
	println(fmt.Sprintf("expecting arr: %d x %d", numCols, numRows))
	game = &Game{}
	game.Rows = numRows * 2
	game.Cols = numCols * 2
	mazeSize := numRows * numCols
	game.Grid = make([]bool, mazeSize*4)
	game.Players = append(
		game.Players,
		&Player{
			Vector{
				Point{float64(startX) * 2, float64(startY) * 2},
				float64(startDirection),
			},
			0.0,
			0.0,
			game,
			make([]bool, mazeSize*4),
		},
	)
	mazePos := 0

	// change scanning strategy- newline chars are a valid possibility in our input maze
	f.Seek(int64(numBytesRead), 0)
	scanner = bufio.NewScanner(f)
	scanner.Split(func(data []byte, isEOF bool) (int, []byte, error) {
		if len(data) == 0 && isEOF {
			return 0, nil, nil
		}
		return len(data), data, nil
	})
	for {
		if !scanner.Scan() {
			break
		}
		mazebufStr := scanner.Text()
		for i, repByte := range mazebufStr {
			row := (mazePos / numCols) * 2
			col := (mazePos % numCols) * 2
			rep := int(repByte)
			if mazePos >= mazeSize {
				game = nil
				err = errors.New(fmt.Sprintf("Data read exceeded expected maze size! Expected: %d Actual: >%d", mazeSize, mazePos+len(mazebufStr)-i))
				return
			}
			if rep == 1 {
				game.Grid[row*game.Cols+col] = true
				game.Grid[row*game.Cols+col+1] = true
				game.Grid[(row+1)*game.Cols+col] = true
				game.Grid[(row+1)*game.Cols+col+1] = true
			} else if rep == 0 {
				game.Grid[row*game.Cols+col] = false
				game.Grid[row*game.Cols+col+1] = false
				game.Grid[(row+1)*game.Cols+col] = false
				game.Grid[(row+1)*game.Cols+col+1] = false
			} else {
				game = nil
				err = errors.New(fmt.Sprintf("Got invalid byte: %d", rep))
				return
			}
			mazePos++
		}
	}
	if mazePos != mazeSize {
		println(fmt.Sprintf("Expected maze size: %d", mazeSize))
		println(fmt.Sprintf("Actual maze size: %d", mazePos))
		game = nil
		err = errors.New("Cannot continue with size discrepancy.")
		return
	}

	return

}

func (game *Game) isEndpoint(p Point) bool {
	index := int(p.Y)*game.Cols + int(p.X)
	if index < 0 {
		return false
	}
	if index >= len(game.Grid) {
		return false
	}
	return game.Grid[index]
}

// see: https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection
func segmentIntersection(p1, p2, p3, p4 Point) (intersection Point) {

	d := (p1.X-p2.X)*(p3.Y-p4.Y) - (p1.Y-p2.Y)*(p3.X-p4.X)

	if math.Abs(d) < 0.000001 {
		intersection.X = math.NaN()
		return
	}

	tn := (p1.X-p3.X)*(p3.Y-p4.Y) - (p1.Y-p3.Y)*(p3.X-p4.X)
	if math.Signbit(d) != math.Signbit(tn) || math.Abs(d) < math.Abs(tn) {
		intersection.X = math.NaN()
		return
	}
	un := (p1.X-p3.X)*(p1.Y-p2.Y) - (p1.Y-p3.Y)*(p1.X-p2.X)
	if math.Signbit(d) != math.Signbit(un) || math.Abs(d) < math.Abs(un) {
		intersection.X = math.NaN()
		return
	}

	intersection = Point{}
	intersection.X = ((p1.X*p2.Y-p1.Y*p2.X)*(p3.X-p4.X) - (p1.X-p2.X)*(p3.X*p4.Y-p3.Y*p4.X)) / d
	intersection.Y = ((p1.X*p2.Y-p1.Y*p2.X)*(p3.Y-p4.Y) - (p1.Y-p2.Y)*(p3.X*p4.Y-p3.Y*p4.X)) / d

	return

}

func getDist(p1, p2 Point) float64 {
	xx := float64(p1.X - p2.X)
	yy := float64(p1.Y - p2.Y)
	return math.Sqrt(xx*xx + yy*yy)
}

func (p *Player) move() {
	px := p.Velocity * math.Sin(p.Dir)
	py := p.Velocity * math.Cos(p.Dir)
	pp := Point{
		p.X + px*10,
		p.Y,
	}
	if !p.Game.isEndpoint(pp) {
		p.X += px
	} else {
		p.X -= px
	}
	pp = Point{
		p.X,
		p.Y + py*10,
	}
	if !p.Game.isEndpoint(pp) {
		p.Y += py
	} else {
		p.Y -= py
	}
	p.Dir = math.Mod(p.Dir+p.RotVel+TWOPI, TWOPI)
	i := int(p.Y)*p.Game.Cols + int(p.X)
	if !(i < 0 || i >= len(p.Visited)) {
		p.Visited[int(p.Y)*p.Game.Cols+int(p.X)] = true
	}
}

func (p *Player) rotate(rot float64) *Vector {
	return &Vector{Point{p.X, p.Y}, math.Mod(p.Dir+rot+TWOPI, TWOPI)}
}

func (p *Player) isVisited(point Point) bool {
	i := int(point.Y)*p.Game.Cols + int(point.X)
	if i < 0 || i >= len(p.Visited) {
		return false
	}
	return p.Visited[i]
}

func (vec *Vector) advance(dist float64) (res *Vector) {
	res = &Vector{
		Point{
			vec.X + math.Sin(vec.Dir)*dist,
			vec.Y + math.Cos(vec.Dir)*dist,
		},
		vec.Dir,
	}
	return
}

func ULBoundary(p Point) Point {
	return Point{X: math.Floor(p.X) - MICRO, Y: math.Floor(p.Y) - MICRO}
}

func URBoundary(p Point) Point {
	return Point{X: math.Ceil(p.X) + MICRO, Y: math.Floor(p.Y) - MICRO}
}

func LRBoundary(p Point) Point {
	return Point{X: math.Ceil(p.X) + MICRO, Y: math.Ceil(p.Y) + MICRO}
}

func LLBoundary(p Point) Point {
	return Point{X: math.Floor(p.X) - MICRO, Y: math.Ceil(p.Y) + MICRO}
}

func doSingleRay(game *Game, vec *Vector) Point {
	var p1, p2, p3, p4 Point
	p1 = vec.Point
	advanced := vec.advance(255)
	p2 = advanced.Point
	startIx := 0
	if vec.Dir < THREEO {
		startIx = 1
	} else if vec.Dir < SIXO {
		startIx = 2
	} else if vec.Dir < NINEO {
		startIx = 3
	} else {
		startIx = 0
	}
	for {
		isUpdated := false
		for i := 0; i < 4; i++ {
			switch (startIx + i) % 4 {
			case 0:
				p3 = LLBoundary(p1)
				p4 = ULBoundary(p1)
			case 1:
				p3 = ULBoundary(p1)
				p4 = URBoundary(p1)
			case 2:
				p3 = URBoundary(p1)
				p4 = LRBoundary(p1)
			case 3:
				p3 = LRBoundary(p1)
				p4 = LLBoundary(p1)
			default:
				return Point{math.NaN(), math.NaN()}
			}
			intersection := segmentIntersection(p1, p2, p3, p4)
			if !math.IsNaN(intersection.X) {
				if game.isEndpoint(intersection) {
					return intersection
				}
				isUpdated = true
				p1 = intersection
				break
			}
		}
		if !isUpdated {
			break
		}
	}
	return Point{math.NaN(), math.NaN()}
}

func doRaytrace(completedCh chan int32, rayCh chan Ray, screen *Screen, segmentIx int32, game *Game, player *Player) {
	defer func() {
		completedCh <- segmentIx
	}()
	segment := screen.Segments[segmentIx]
	width := segment.W
	indexStart := segment.X
	indexStop := indexStart + width
	isfilled := make([]bool, width)
	dists := make([]float64, width)
	rotated := make([]*Vector, width)
	for i := indexStart; i < indexStop; i++ {
		col := i - indexStart
		rotated[col] = player.rotate(screen.HAngles[i])
		endpoint := doSingleRay(game, rotated[col])
		if math.IsNaN(endpoint.X) {
			continue
		}
		isfilled[col] = true
		dists[col] = getDist(player.Point, endpoint)
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
					x := byte(255) - byte(math.Log(xDist+yDist*2)*45)
					data[colStart] = x
					data[colStart+1] = x
					data[colStart+2] = x
					continue
				}
			}
			if isUpper {
				if ceilDist < 256 {
					rayvec := rotated[col]
					ceilpoint1 := rayvec.advance(ceilDist)
					if player.isVisited(ceilpoint1.Point) {
						data[colStart] = 0
						data[colStart+1] = 0
						data[colStart+2] = 255
						continue
					}
				}
			} else {
				if floorDist < 256 {
					x := math.Log(floorDist)
					data[colStart] = byte(17 - x*3)
					data[colStart+1] = byte(17 - x*3)
					data[colStart+2] = byte(85 - x*15)
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

	println("In DoMaze()")

	FRAMESLEEP := int64(math.Floor(1.0 / FRAMERATE * 1e9))
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
			println(fmt.Sprintf("FPS: %d", numFrames))
			intervalStartSec = startTime
			numFrames = 0
		}
		select {
		case msg := <-recvCh:
			if msg == End {
				return
			}
			println("you goofed")
		default:
			var err error
			err = renderer.SetDrawColor(0, 0, 0, 255)
			if err != nil {
				println(fmt.Sprintf("Got error on SetDrawColor(): %s", err))
				return
			}
			err = renderer.Clear()
			if err != nil {
				println(fmt.Sprintf("Got error on Clear(): %s", err))
				return
			}
			numCompleted = 0
			if len(completedCh) != 0 {
				println("completedCh should be empty")
				return
			}
			for i := int32(0); i < NUMWORKERS; i++ {
				go doRaytrace(completedCh, rayCh, screen, i, game, player)
			}
			pixels, _, err := targetTexture.Lock(nil)
			if err != nil {
				println(fmt.Sprintf("Got err from Lock(): %s", err))
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
			//println("got past loop")
			targetTexture.Unlock()
			renderer.Copy(targetTexture, nil, nil)
			renderer.Present()
		}
		dur := time.Now().UnixNano() - startTime
		if dur > 0 {
			time.Sleep(time.Duration(FRAMESLEEP-dur) * time.Nanosecond)
		}
		numFrames++
		game.Players[0].move()
	}

}
