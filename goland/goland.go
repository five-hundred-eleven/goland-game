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
	FRAMERATE          = 16666666
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
	MICRO              = 0.0000001
	SHEARFACTOR        = 1.05
	WINDOWWIDTH  int32 = 640
	WINDOWHEIGHT int32 = 480
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
	Players    []*Vector
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
	game.Rows = numRows
	game.Cols = numCols
	mazeSize := numRows * numCols
	game.Grid = make([]bool, mazeSize)
	game.Players = append(game.Players, &Vector{Point{float64(startX), float64(startY)}, float64(startDirection)})
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
			rep := int(repByte)
			if mazePos >= mazeSize {
				game = nil
				err = errors.New(fmt.Sprintf("Data read exceeded expected maze size! Expected: %d Actual: >%d", mazeSize, mazePos+len(mazebufStr)-i))
				return
			}
			if rep == 1 {
				game.Grid[mazePos] = true
			} else if rep == 0 {
				game.Grid[mazePos] = false
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

func (vec *Vector) rotate(rot float64) {
	vec.Dir = math.Mod(vec.Dir + rot, TWOPI)
}

func (vec *Vector) rotated(rot float64) (*Vector) {
	return &Vector{Point{vec.X, vec.Y}, math.Mod(vec.Dir + rot, TWOPI)}
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

func ray(game *Game, vec *Vector) Point {
	if vec.Dir < THREEO {
		return rayLR(game, vec)
	} else if vec.Dir < SIXO {
		return rayUR(game, vec)
	} else if vec.Dir < NINEO {
		return rayUL(game, vec)
	}
	return rayLL(game, vec)
}

func rayLR(game *Game, vec *Vector) Point {
	p1 := Point{vec.X, vec.Y}
	advanced := vec.advance(150)
	p2 := Point{advanced.X, advanced.Y}
	for {
		p3 := LLBoundary(p1)
		p4 := LRBoundary(p1)
		intersection := segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		p3 = URBoundary(p1)
		intersection = segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		break
	}
	return Point{math.NaN(), math.NaN()}
}

func rayUR(game *Game, vec *Vector) Point {
	p1 := Point{vec.X, vec.Y}
	advanced := vec.advance(150)
	p2 := Point{advanced.X, advanced.Y}
	for {
		p3 := ULBoundary(p1)
		p4 := URBoundary(p1)
		intersection := segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		p3 = LRBoundary(p1)
		intersection = segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		break
	}
	return Point{math.NaN(), math.NaN()}
}

func rayUL(game *Game, vec *Vector) Point {
	p1 := Point{vec.X, vec.Y}
	advanced := vec.advance(150)
	p2 := Point{advanced.X, advanced.Y}
	for {
		p3 := LLBoundary(p1)
		p4 := ULBoundary(p1)
		intersection := segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		p3 = URBoundary(p1)
		intersection = segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		break
	}
	return Point{math.NaN(), math.NaN()}
}

func rayLL(game *Game, vec *Vector) Point {
	p1 := Point{vec.X, vec.Y}
	advanced := vec.advance(150)
	p2 := Point{advanced.X, advanced.Y}
	for {
		p3 := ULBoundary(p1)
		p4 := LLBoundary(p1)
		intersection := segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		p3 = LRBoundary(p1)
		intersection = segmentIntersection(p1, p2, p3, p4)
		if !math.IsNaN(intersection.X) {
			if game.isEndpoint(intersection) {
				return intersection
			}
			p1 = intersection
			continue
		}
		break
	}
	return Point{math.NaN(), math.NaN()}
}

func doRaytrace(completedCh chan int32, screen *Screen, segmentIx int32, game *Game, player *Vector) {
	defer func() {
		completedCh <- segmentIx
	}()
	//println(fmt.Sprintf("doRaytrace(): indexstart: %d, indexStop: %d, dir: %f", indexStart, indexStop, player.Dir))
	texture := screen.SegTextures[segmentIx]
	segment := screen.Segments[segmentIx]
	width := segment.W
	indexStart := segment.X
	indexStop := indexStart + width
	isfilled := make([]bool, width)
	endpoints := make([]Point, width)
	for i := indexStart; i < indexStop; i++ {
		rayvec := player.rotated(screen.HAngles[i])
		endpoint := ray(game, rayvec)
		if math.IsNaN(endpoint.X) {
			continue
		}
		isfilled[i-indexStart] = true
		endpoints[i-indexStart] = endpoint
	}
	pixels, _, err := texture.Lock(screen.SegMask)
	if err != nil {
		println(fmt.Sprintf("Got error on Lock(): %s", err))
		return
	}
    //println(fmt.Sprintf("Got pixels: %d, expected: %d", len(pixels), width * screen.Height * 4))
	for row := int32(0); row < screen.Height; row++ {
		rowStart := row * width * 4
		for col := int32(0); col < width; col++ {
			colStart := rowStart + col*4
            /*
            if colStart >= int32(len(pixels)) {
                println(fmt.Sprintf("Got error index: row: %d, col: %d, width: %d, rowStart: %d, colStart: %d, len: %d, indexStart: %d", row, col, width, rowStart, colStart, len(pixels), indexStart))
                return
            }
            */
			if isfilled[col] {
				// TODO don't do this repeatedly
				// should probably refactor this whole function...
				dist := getDist(player.Point, endpoints[col])
				//x := math.Abs(math.Cos(screen.VAngles[row])*dist)
				if 5.0 > math.Abs(math.Sin(screen.VAngles[row])*dist) {
					pixels[colStart] = 127
					pixels[colStart+1] = 127
					pixels[colStart+2] = 127
					pixels[colStart+3] = 255
				} else {
					pixels[colStart] = 0
					pixels[colStart+1] = 0
					pixels[colStart+2] = 0
					pixels[colStart+3] = 255
				}
			} else {
				pixels[colStart] = 0
				pixels[colStart+1] = 0
				pixels[colStart+2] = 0
				pixels[colStart+3] = 255
			}
		}
	}
	texture.Unlock()
}

func DoMaze(recvCh chan int, sendCh chan int, screen *Screen, game *Game) {

	defer func() {
		sendCh <- End
	}()

	println("In DoMaze()")

	renderer := screen.Renderer
	//surfaceTexture := screen.SurfaceTexture
	//targetTexture := screen.TargetTexture
	segTextures := screen.SegTextures
	segments := screen.Segments
	//targetMask := screen.TargetMask
	segMask := screen.SegMask

	widthF := float64(screen.Width)
	depthF := float64(screen.Depth)
	screen.HAngles = make([]float64, screen.Width)
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

	heightF := float64(screen.Height)
	screen.VAngles = make([]float64, screen.Height)
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
			/*
				            err := renderer.SetDrawColor(0, 0, 0, 255)
				            if err != nil {
				                println(fmt.Sprintf("Got error on SetDrawColor(): %s", err))
				                return
				            }
							err = renderer.Clear()
							if err != nil {
								println(fmt.Sprintf("Got error on Clear(): %s", err))
								return
							}
			*/
            //var err error
			numCompleted = 0
            /*
            err := renderer.SetRenderTarget(targetTexture)
			if err != nil {
				println(fmt.Sprintf("Got error in SetRenderTarget: %s", err))
				return
			}
            */
			if len(completedCh) != 0 {
				println("completedCh should be empty")
				return
			}
			for i := int32(0); i < NUMWORKERS; i++ {
				go doRaytrace(completedCh, screen, i, game, player)
			}
			for {
				indexCompleted := <-completedCh
				numCompleted++
				err := renderer.Copy(segTextures[indexCompleted], segMask, segments[indexCompleted])
				if err != nil {
					println(fmt.Sprintf("Got error in Copy(): %s", err))
					return
				}
				if numCompleted >= NUMWORKERS {
					if len(completedCh) != 0 {
						println("completedCh should be empty")
					}
					break
				}
			}
            /*
            err = renderer.SetRenderTarget(surfaceTexture)
			if err != nil {
				println(fmt.Sprintf("Got error in SetRenderTarget: %s", err))
				return
			}
			err = renderer.Copy(targetTexture, targetMask, targetMask)
			if err != nil {
				println(fmt.Sprintf("Got error in Copy(): %s", err))
				return
			}
            */
			renderer.Present()
		}
		dur := time.Now().UnixNano() - startTime
		if dur > 0 {
			time.Sleep(time.Duration(FRAMERATE-dur) * time.Nanosecond)
		}
		numFrames++
		game.Players[0].rotate(0.01)
	}

}
