package main

import (
	"fmt"
	"os"

	"github.com/five-hundred-eleven/goland-game/goland"
	"github.com/veandco/go-sdl2/sdl"
)

func main() {
	mazeFilePath := "maze.json"
	if len(os.Args) > 1 {
		mazeFilePath = os.Args[1]
	}

	game, err := goland.NewGameFromFilename(mazeFilePath)
	if err != nil {
		fmt.Printf("Got error constructing game: %s\n", err)
		return
	}

	fmt.Printf("Got game\n")
	fmt.Printf("Got player %f %f %f\n", game.Players[0].X, game.Players[0].Y, game.Players[0].Theta)
	//fmt.Printf("Got surfaces: len: %d\n", len(game.Surfaces))
	//fmt.Printf("First surface: %f %f %f %f\n", game.Surfaces[0].P1.X, game.Surfaces[0].P1.Y, game.Surfaces[0].P2.X, game.Surfaces[0].P2.Y)
	//fmt.Sprintf("Got game: %d x %d", game.Cols, game.Rows)

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Printf("Unable to init! %s\n", err)
		return
	}
	defer sdl.Quit()

	screen := &goland.Screen{}
	// width, height := window.GetSize()
	screen.Width = goland.WINDOWWIDTH
	screen.Height = goland.WINDOWHEIGHT
	// this is a constant for right triangles
	screen.Depth = screen.Width / 2

	screen.Window, err = sdl.CreateWindow(
		"Goland!",
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		goland.WINDOWWIDTH,
		goland.WINDOWHEIGHT,
		0,
	)
	if err != nil {
		fmt.Printf("Unable to create goland! %s\n", err)
		return
	}
	defer screen.Window.Destroy()

	screen.Renderer, err = sdl.CreateRenderer(screen.Window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		fmt.Printf("Got error in CreateRenderer(): %s\n", err)
		return
	}
	defer screen.Renderer.Destroy()

	surface, err := screen.Window.GetSurface()
	if err != nil {
		fmt.Printf("Got error in GetSurface(): %s\n", err)
		return
	}
	screen.Format = surface.Format
	err = surface.SetBlendMode(sdl.BLENDMODE_ADD)
	if err != nil {
		fmt.Printf("Got error in SetBlendMode(): %s\n", err)
		return
	}
	screen.TargetTexture, err = screen.Renderer.CreateTexture(
		screen.Format.Format,
		sdl.TEXTUREACCESS_STREAMING,
		goland.WINDOWWIDTH,
		goland.WINDOWHEIGHT,
	)
	if err != nil {
		fmt.Printf("Got error in CreateTexture(): %s\n", err)
		return
	}
	defer screen.TargetTexture.Destroy()
	screen.SurfaceTexture = screen.Renderer.GetRenderTarget()
	if screen.SurfaceTexture != nil {
		fmt.Printf("Got error in GetRenderTarget()\n")
		return
	}

	segWidth := goland.WINDOWWIDTH / goland.NUMWORKERS
	screen.SegTextures = make([]*sdl.Texture, goland.NUMWORKERS)
	screen.Segments = make([]*sdl.Rect, goland.NUMWORKERS)
	for i := int32(0); i < goland.NUMWORKERS; i++ {
		screen.Segments[i] = &sdl.Rect{X: segWidth * i, Y: 0, W: segWidth, H: goland.WINDOWHEIGHT}
	}
	screen.TargetMask = &sdl.Rect{X: 0, Y: 0, W: goland.WINDOWWIDTH, H: goland.WINDOWHEIGHT}
	screen.SegMask = &sdl.Rect{X: 0, Y: 0, W: segWidth, H: goland.WINDOWHEIGHT}

	sendCh := make(chan int)
	recvCh := make(chan int)
	go goland.DoMaze(sendCh, recvCh, screen, game)

	go func() {
		defer func() {
			sendCh <- goland.End
		}()
		for {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch e := event.(type) {
				case *sdl.QuitEvent:
					fmt.Printf("Got quit event!\n")
					return
				case *sdl.KeyboardEvent:
					if e.Type == sdl.KEYDOWN {
						switch e.Keysym.Scancode {
						case sdl.SCANCODE_W:
							game.Players[0].Velocity = goland.VELOCITY
						case sdl.SCANCODE_S:
							game.Players[0].Velocity = -goland.VELOCITY
						case sdl.SCANCODE_A:
							game.Players[0].RotVel = -goland.ROTVEL
						case sdl.SCANCODE_D:
							game.Players[0].RotVel = goland.ROTVEL
						}
					} else if e.Type == sdl.KEYUP {
						switch e.Keysym.Scancode {
						case sdl.SCANCODE_W:
							game.Players[0].Velocity = 0.0
						case sdl.SCANCODE_S:
							game.Players[0].Velocity = 0.0
						case sdl.SCANCODE_A:
							game.Players[0].RotVel = 0.0
						case sdl.SCANCODE_D:
							game.Players[0].RotVel = 0.0
						}
					} else {
						// println("Got unrecognized keyboard event")
					}
				default:
					// println(fmt.Sprintf("Got unrecognized event! %d", event.GetType()))
				}
			}
		}
	}()

	for {
		msg := <-recvCh
		if msg == goland.End {
			fmt.Printf("Goland got destroyed!\n")
			break
		} else {
			fmt.Printf("Got unrecognized message??\n")
		}
	}
}
