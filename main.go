package main

import (
	"fmt"
	"github.com/five-hundred-eleven/goland-game/goland"
	"github.com/veandco/go-sdl2/sdl"
	"os"
)

const ()

func main() {

	mazeFilePath := "maze.txt"
	if len(os.Args) > 1 {
		mazeFilePath = os.Args[1]
	}

	game, err := goland.NewGameFromFilename(mazeFilePath)
	if err != nil {
		fmt.Sprintln("Got error constructing game: %s", err)
		return
	}

	println(fmt.Sprintf("Got game: %d x %d", game.Cols, game.Rows))

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		fmt.Sprintln("Unable to init! %s", err)
		return
	}
	defer sdl.Quit()

	screen := &goland.Screen{}
	//width, height := window.GetSize()
	screen.Width = goland.WINDOWWIDTH
	screen.Height = goland.WINDOWHEIGHT
	// this is a constant for right triangles
	screen.Depth = screen.Width / 2

	screen.Window, err = sdl.CreateWindow("Goland!", sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED, goland.WINDOWWIDTH, goland.WINDOWHEIGHT, 0)
	if err != nil {
		println(fmt.Sprintf("Unable to create goland! %s", err))
		return
	}
	defer screen.Window.Destroy()

	screen.Renderer, err = sdl.CreateRenderer(screen.Window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		println(fmt.Sprintf("Got error in CreateRenderer(): %s", err))
		return
	}
	defer screen.Renderer.Destroy()

	surface, err := screen.Window.GetSurface()
	if err != nil {
		println(fmt.Sprintf("Got error in GetSurface(): %s", err))
		return
	}
	screen.Format = surface.Format
	err = surface.SetBlendMode(sdl.BLENDMODE_ADD)
	if err != nil {
		println(fmt.Sprintf("Got error in SetBlendMode(): %s", err))
		return
	}
	screen.TargetTexture, err = screen.Renderer.CreateTexture(screen.Format.Format, sdl.TEXTUREACCESS_STREAMING, goland.WINDOWWIDTH, goland.WINDOWHEIGHT)
	if err != nil {
		println(fmt.Sprintf("Got error in CreateTexture(): %s", err))
		return
	}
	defer screen.TargetTexture.Destroy()
	screen.SurfaceTexture = screen.Renderer.GetRenderTarget()
	if screen.SurfaceTexture != nil {
		println(fmt.Sprintf("Got error in GetRenderTarget()"))
		return
	}
	/*
	   surfaceTexture, err := screen.Renderer.CreateTextureFromSurface(surface)
	   if err != nil {
	       println(fmt.Sprintf("Got error in CreateTextureFromSurface(): %s", err))
	       return
	   }
	   defer surfaceTexture.Destroy()
	   err = screen.Renderer.SetRenderTarget(screen.surfaceTexture)
	   if err != nil {
	       println(fmt.Sprintf("Got error in SetRenderTarget(): %s", err))
	       return
	   }
	*/

	segWidth := goland.WINDOWWIDTH / goland.NUMWORKERS
	screen.SegTextures = make([]*sdl.Texture, goland.NUMWORKERS)
	screen.Segments = make([]*sdl.Rect, goland.NUMWORKERS)
	for i := int32(0); i < goland.NUMWORKERS; i++ {
		/*
				screen.SegTextures[i], err = screen.Renderer.CreateTexture(screen.Format.Format, sdl.TEXTUREACCESS_STREAMING, goland.WINDOWWIDTH, goland.WINDOWHEIGHT)
				if err != nil {
					println(fmt.Sprintf("Got error in CreateTexture(): %s", err))
					return
				}
		        println(fmt.Sprintf("Got segment texture: %d x %d", segWidth, goland.WINDOWHEIGHT))
				defer screen.SegTextures[i].Destroy()
				err = screen.SegTextures[i].SetBlendMode(sdl.BLENDMODE_NONE)
				if err != nil {
					println(fmt.Sprintf("Got error in SetBlendMode(): %s", err))
					return
				}
		*/
		screen.Segments[i] = &sdl.Rect{segWidth * i, 0, segWidth, goland.WINDOWHEIGHT}
	}
	screen.TargetMask = &sdl.Rect{0, 0, goland.WINDOWWIDTH, goland.WINDOWHEIGHT}
	screen.SegMask = &sdl.Rect{0, 0, segWidth, goland.WINDOWHEIGHT}

	sendCh := make(chan int)
	recvCh := make(chan int)
	go goland.DoMaze(sendCh, recvCh, screen, game)

	go func() {
		defer func() {
			recvCh <- goland.End
		}()
		for {
			for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
				switch e := event.(type) {
				case *sdl.QuitEvent:
					println("Got quit event!")
					return
				case *sdl.KeyboardEvent:
					if e.Type == sdl.KEYDOWN {
						switch e.Keysym.Scancode {
						case sdl.SCANCODE_W:
							game.Players[0].Velocity = 0.3
							break
						case sdl.SCANCODE_S:
							game.Players[0].Velocity = -0.3
							break
						case sdl.SCANCODE_A:
							game.Players[0].RotVel = -0.1
							break
						case sdl.SCANCODE_D:
							game.Players[0].RotVel = 0.1
							break
						}
					} else if e.Type == sdl.KEYUP {
						switch e.Keysym.Scancode {
						case sdl.SCANCODE_W:
						case sdl.SCANCODE_S:
							game.Players[0].Velocity = 0.0
							break
						case sdl.SCANCODE_A:
						case sdl.SCANCODE_D:
							game.Players[0].RotVel = 0.0
							break
						}
					} else {
						println("Got unrecognized keyboard event")
					}
				default:
					println(fmt.Sprintf("Got unrecognized event! %d", event.GetType()))
				}
			}
		}
	}()

	for {
		msg := <-recvCh
		if msg == goland.End {
			println("Goland got destroyed!")
			break
		} else {
			println("Got unrecognized message??")
		}
	}

}
